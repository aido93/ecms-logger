package ECMSLogger

import (
	"time"
)

type AccessRecord struct {
	Time       time.Time `db:"time" json:"time"`
	ClientTime time.Time `db:"client_time" json:"clientTime"`
	// from context
	RedisDurationUs   uint64  `db:"redis_duration_us" json:"redisDurationUs"`
	Host              string  `db:"host" json:"host"`
	Method            string  `db:"method" json:"method"`
	RequestURI        string  `db:"request_uri" json:"requestURI"`
	Version           string  `db:"version" json:"version"`
	Category          string  `db:"category" json:"category"`
	Subject           string  `db:"subject"` //URI without category
	RemoteAddr        string  `db:"remote_addr" json:"remoteAddr"`
	ContentLength     int64   `db:"content_length" json:"contentLength"`
	Continent         string  `db:"continent" json:"continent"`
	Country           string  `db:"country" json:"country"`
	IsoCountry        string  `db:"iso_country" json:"isoCountry"`
	City              string  `db:"city" json:"city"`
	Latitude          float64 `db:"latitude" json:"latitude"`
	Longitude         float64 `db:"longitude" json:"longitude"`
	AccuracyRadius    uint16  `db:"accuracy_radius" json:"accuracyRadius"`
	Timezone          string  `db:"timezone" json:"timezone"`
	Subdivision       string  `db:"subdivision" json:"subdivision"`
	IsInEuropeanUnion bool    `db:"eu_member" json:"euMember"`
	DurationUs        uint64  `db:"duration_us" json:"durationUs"`
	DBDurationUs      uint64  `db:"db_duration_us" json:"dbDurationUs"`
	// from headers
	OS      string `db:"os" json:"os"`
	Browser string `db:"browser" json:"browser"`
	Width   uint32 `db:"width" json:"width"`
	Height  uint32 `db:"height" json:"height"`
	// from request
	User             string `db:"user" json:"user"`
	UserAgent        string `db:"user_agent" json:"userAgent"`
	Source           string `db:"source" json:"source"`
	Target           string `db:"target" json:"target"`
	Params           string `db:"params" json:"params"` //URL params in JSON format
	ClientName       string `db:"client_name" json:"clientName"`
	ClientBranch     string `db:"client_branch" json:"clientBranch"`
	ClientCommitHash string `db:"client_commit_hash" json:"clientCommitHash"`
	ClientTag        string `db:"client_tag" json:"clientTag"`
	// from response
	Status         uint16 `db:"status" json:"status"`
	Response       string `db:"response" json:"response"`
	ResponseLength uint64 `db:"response_length" json:"responseLength"`
	Error          string `db:"error" json:"error"`
	// from app
	Region     string `db:"region" json:"region"`
	Location   string `db:"location" json:"location"`
	Branch     string `db:"branch" json:"branch"`
	CommitHash string `db:"commit_hash" json:"commitHash"`
	Tag        string `db:"tag" json:"tag"`
}

func (ar *AccessRecord) Send() {
	records <- *ar
}

func (ar *AccessRecord) GetAvailableFields() []string {
	fields, _ := StructFields(ar, "db", []string{})
	return fields
}
