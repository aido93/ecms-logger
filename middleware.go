package ECMSLogger

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"github.com/oschwald/geoip2-golang"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ClickhouseMiddlewareConfig struct {
	IPSource     string
	MaxMind      *geoip2.Reader
	Logger       Logger
	SessionField string
	Branch       string
	CommitHash   string
	Tag          string
}

var chMiddleware ClickhouseMiddlewareConfig

func (m *ClickhouseMiddlewareConfig) Init(config *Config) {
	m.initMaxMind(&config.MaxMind)
	m.Logger.Init(&config.Clickhouse)
}

func (cm *ClickhouseMiddlewareConfig) initMaxMind(mm *MaxMind) {
	tmp, err := geoip2.Open(mm.DB)
	if err != nil {
		panic(err)
	}
	cm.MaxMind = tmp
	if v, ok := mm.Source["remoteAddr"]; ok && v == "true" {
		cm.IPSource = "remoteAddr"
	} else if v, ok := mm.Source["header"]; ok {
		cm.IPSource = v
	} else {
		panic("Undefined ip source for maxmind")
	}
}

type ClickhouseContext struct {
	echo.Context
	record       AccessRecord
	sess         *sessions.Session
	redisStatus  int
	redisErr     error
	accessStatus int
	accessErr    error
}

func (cs *ClickhouseContext) userNickname(ctx echo.Context) (string, *sessions.Session, int, error) {
	sess, err := aux.Store.Get(ctx.Request(), aux.Session.Cookie)
	if err != nil {
		return "", nil, http.StatusInternalServerError, err
	}
	interfaceId, ok := sess.Values[chMiddleware.SessionField]
	if !ok {
		return "", nil, http.StatusBadRequest, errors.New(chMiddleware.SessionField + " is not found")
	}
	nickname := interfaceId.(string)
	if nickname == "" {
		return "", nil, http.StatusTemporaryRedirect, errors.New(chMiddleware.SessionField + " is empty")
	}
	if twofaId, ok := sess.Values["2fa"]; ok {
		twofa := twofaId.(bool)
		if !twofa {
			return nickname, sess, http.StatusForbidden, errors.New("2FA has not finished")
		}
	}
	return nickname, sess, http.StatusOK, nil
}

func (c *ClickhouseContext) JSON(code int, msg interface{}) error {
	defer func(c *ClickhouseContext) {
		c.record.DurationUs = uint64(time.Since(c.record.Time).Microseconds())
		c.record.ResponseLength = uint64(c.Response().Size)
		c.record.Status = uint16(c.Response().Status)
		c.record.Send()
	}(c)
	switch msg.(type) {
	case error:
		c.record.Error = msg.(error).Error()
		return c.Context.JSON(code, map[string]interface{}{"error": msg.(error).Error()})
	default:
		b, _ := json.Marshal(msg)
		c.record.Response = string(b)
		return c.Context.JSON(code, msg)
	}
}

func (c *ClickhouseContext) JSONOK() error {
	defer func(c *ClickhouseContext) {
		c.record.DurationUs = uint64(time.Since(c.record.Time).Microseconds())
		c.record.ResponseLength = uint64(c.Response().Size)
		c.record.Status = uint16(c.Response().Status)
		c.record.Send()
	}(c)
	return c.Context.JSON(http.StatusOK, map[string]string{})
}

func (c *ClickhouseContext) Render(code int, templ string, vars interface{}) error {
	defer func(c *ClickhouseContext) {
		c.record.DurationUs = uint64(time.Since(c.record.Time).Microseconds())
		c.record.ResponseLength = uint64(c.Response().Size)
		c.record.Status = uint16(c.Response().Status)
		c.record.Send()
	}(c)
	return c.Context.Render(code, templ, vars)
}

func (c *ClickhouseContext) Redirect(code int, url string) error {
	defer func(c *ClickhouseContext) {
		c.record.DurationUs = uint64(time.Since(c.record.Time).Microseconds())
		c.record.ResponseLength = uint64(c.Response().Size)
		c.record.Status = uint16(c.Response().Status)
		c.record.Send()
	}(c)
	return c.Context.Redirect(code, url)
}

func (c *ClickhouseContext) String(code int, str string) error {
	defer func(c *ClickhouseContext) {
		c.record.DurationUs = uint64(time.Since(c.record.Time).Microseconds())
		c.record.ResponseLength = uint64(c.Response().Size)
		c.record.Status = uint16(c.Response().Status)
		c.record.Send()
	}(c)
	return c.Context.String(code, str)
}

func (c *ClickhouseContext) NoContent(code int) error {
	if aux.Server.LogNoContent {
		defer func(c *ClickhouseContext) {
			c.record.DurationUs = uint64(time.Since(c.record.Time).Microseconds())
			c.record.ResponseLength = uint64(c.Response().Size)
			c.record.Status = uint16(c.Response().Status)
			c.record.Send()
		}(c)
	}
	return c.Context.NoContent(code)
}

func (c *ClickhouseContext) SetDBDurationUs(dur time.Duration) {
	c.record.DBDurationUs = uint64(dur.Microseconds())
}

func (c *ClickhouseContext) SetSource(source string) {
	c.record.Source = source
}

func (c *ClickhouseContext) SetTarget(target string) {
	c.record.Target = target
}

func (c *ClickhouseContext) Session() *sessions.Session {
	return c.sess
}

func (c *ClickhouseContext) Err() error {
	return c.redisErr
}

func (c *ClickhouseContext) Status() int {
	return c.redisStatus
}

func (c *ClickhouseContext) Nickname() string {
	return c.record.User
}

func (cc *ClickhouseContext) getAccessRecord() {
	cc.record.Time = time.Now()
	cc.record.Region = aux.Server.Region
	cc.record.Location = aux.Server.Location
	cc.record.Branch = chMiddleware.Branch
	cc.record.CommitHash = chMiddleware.CommitHash
	cc.record.Tag = chMiddleware.Tag
	slug, sess, status, redisErr := cc.userNickname(cc.Context)
	cc.sess = sess
	cc.redisStatus = status
	cc.redisErr = redisErr
	if sess != nil {
		cc.record.User = slug
	}
	cc.record.RedisDurationUs = uint64(time.Since(cc.record.Time).Microseconds())
	req := cc.Context.Request()
	cc.record.Host = req.Host
	cc.record.Method = req.Method
	cc.record.RequestURI = req.RequestURI
	splitted := strings.Split(req.RequestURI[1:len(req.RequestURI)], "/")
	if len(splitted) > 0 {
		cc.record.Version = splitted[0]
	}
	if len(splitted) > 1 {
		cc.record.Category = splitted[1]
	}
	if len(splitted) > 2 {
		cc.record.Subject = strings.Split(strings.Join(splitted[2:len(splitted)], "/"), "?")[0]
	}
	// cc.record.User is empty when not authorized or anonymous
	// cc.record.Category should always be
	// cc.record.Subject may be empty
	status, err := checkAccess(cc.record.User, cc.record.Category, cc.record.Subject, cc.record.Method)
	cc.accessStatus = status
	cc.accessErr = err
	cc.record.ContentLength = req.ContentLength
	cc.record.UserAgent = req.Header.Get("User-Agent")
	cc.record.ClientName = req.Header.Get("X-Client-Name")
	cc.record.ClientBranch = req.Header.Get("X-Client-Branch")
	cc.record.ClientCommitHash = req.Header.Get("X-Client-Commit-Hash")
	cc.record.ClientTag = req.Header.Get("X-Client-Tag")
	cc.record.OS = req.Header.Get("X-OS")
	cc.record.Browser = req.Header.Get("X-Browser")
	params := cc.Context.QueryParams()
	p, err := json.Marshal(params)
	if err != nil {
		log.Warning("Cannot marhal params: ", err)
	}
	cc.record.Params = string(p)
	w := req.Header.Get("X-Width")
	if w != "" {
		if width, err := strconv.ParseUint(w, 10, 64); err == nil {
			cc.record.Width = uint32(width)
		}
	}
	h := req.Header.Get("X-Height")
	if h != "" {
		if height, err := strconv.ParseUint(h, 10, 64); err == nil {
			cc.record.Height = uint32(height)
		}
	}
	ipaddr := ""
	if chMiddleware.IPSource != "remoteAddr" {
		if chMiddleware.IPSource != "" {
			ipaddr = req.Header.Get(chMiddleware.IPSource)
		}
	} else {
		ipaddr = req.RemoteAddr
	}
	if ipaddr != "" {
		ip := net.ParseIP(ipaddr)
		if chMiddleware.MaxMind != nil {
			record, err := chMiddleware.MaxMind.City(ip)
			if err == nil {
				cc.record.Country = record.Country.Names["en"]
				cc.record.IsoCountry = record.Country.IsoCode
				cc.record.City = record.City.Names["en"]
				cc.record.Longitude = record.Location.Longitude
				cc.record.Latitude = record.Location.Latitude
				cc.record.Timezone = record.Location.TimeZone
				cc.record.Continent = record.Continent.Names["en"]
				cc.record.AccuracyRadius = record.Location.AccuracyRadius
				cc.record.IsInEuropeanUnion = record.Country.IsInEuropeanUnion
				loc, _ := time.LoadLocation(cc.record.Timezone)
				cc.record.ClientTime = cc.record.Time.In(loc)
				subd := record.Subdivisions
				if len(subd) > 0 {
					cc.record.Subdivision = subd[0].Names["en"]
				}
			} else {
				log.Warning("Cannot determine ip location: ", err)
			}
		}
		cc.record.RemoteAddr = ipaddr
	}
}

func ClickhouseMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := &ClickhouseContext{c, logger.AccessRecord{}, nil, http.StatusOK, nil, http.StatusOK, nil}
		cc.getAccessRecord()
		if err := next(cc); err != nil {
			cc.record.Error = err.Error()
			cc.record.DurationUs = uint64(time.Since(cc.record.Time).Microseconds())
			cc.record.ResponseLength = uint64(cc.Response().Size)
			cc.record.Status = uint16(cc.Response().Status)
			cc.record.Send()
			return err
		}
		return nil
	}
}

func ClickhouseHTTPErrorHandler(err error, c echo.Context) {
	he, ok := err.(*echo.HTTPError)
	if ok {
		if he.Internal != nil {
			if herr, ok := he.Internal.(*echo.HTTPError); ok {
				he = herr
			}
		}
	} else {
		he = &echo.HTTPError{
			Code:    http.StatusInternalServerError,
			Message: http.StatusText(http.StatusInternalServerError),
		}
	}
	code := he.Code
	//m, _ := json.Marshal(he.Message)
	//msg=string(m)
	if !c.Response().Committed {
		var err1 error
		if c.Request().Method == http.MethodHead { // Issue #608
			err1 = c.NoContent(he.Code)
		} else {
			msg := he.Message.(string)
			cc := &ClickhouseContext{c, logger.AccessRecord{}, nil, http.StatusOK, nil, http.StatusOK, nil}
			cc.getAccessRecord()
			cc.record.Error = msg
			defer func(c *ClickhouseContext) {
				c.record.DurationUs = uint64(time.Since(c.record.Time).Microseconds())
				c.record.ResponseLength = uint64(c.Response().Size)
				c.record.Status = uint16(c.Response().Status)
				c.record.Send()
			}(cc)
			err1 = c.JSON(code, map[string]interface{}{"error": msg})
		}
		if err1 != nil {
			c.Logger().Error(err1)
		}
	}
}
