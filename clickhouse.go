package ECMSLogger

import (
	"fmt"
	_ "github.com/ClickHouse/clickhouse-go"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

type Logger struct {
	chWriter        *sqlx.DB
	chInsertQuery   string
	logTable        string
	availableFields []string
}

var records chan AccessRecord

func formConnectionString(c *Connection) string {
	address_template := "tcp://%s:%d?username=%s&password=%s&database=%s&write_timeout=%d&debug=%v"
	address := fmt.Sprintf(address_template, c.Host, c.Port, c.User, c.Password, c.DB, c.Timeout.Seconds(), c.Debug)
	if len(c.AltHosts) > 0 {
		address += ("&alt_hosts=" + strings.Join(c.AltHosts, ","))
	}
	return address
}

func (l *Logger) Init(cs *ClickhouseSettings) {
	if cs.Reserve != nil {
		maxSize = ParseSize(cs.Reserve.Rotate.MaxSize)
		if maxSize == 0 {
			panic("Wrong maxSize")
		}
		if err := CheckTouch(cs.Reserve.Dir); err != nil {
			panic("Cannot touch in " + cs.Reserve.Dir + ": " + err.Error())
		}
	}
	l.logTable = cs.Table
	if l.logTable == "" {
		panic("Log table has empty name")
	}
	address := formConnectionString(&cs.Connection)
	conn, err := sqlx.Open("clickhouse", address)
	if err != nil {
		panic("failed to open clickhouse database on read: " + err.Error())
	}
	i := 1
	for i < 10 {
		err = conn.Ping()
		if err == nil {
			break
		} else {
			if cs.Connection.Debug {
				log.Info("clickhouse database is not ready. waiting...")
			}
			time.Sleep(time.Duration(i) * time.Second)
		}
		i += 1
	}
	if i >= 10 {
		panic("Clickhouse cannot ping: " + err.Error())
	}
	if cs.Connection.IdleLimit != 0 {
		conn.SetMaxIdleConns(cs.Connection.IdleLimit)
	}
	if cs.Connection.ConnLimit != 0 {
		conn.SetMaxOpenConns(cs.Connection.ConnLimit)
	}
	_, err = l.chWriter.Exec(`
		CREATE TABLE IF NOT EXISTS ` + l.logTable + ` (
			time			   DateTime,
			client_time	  	   DateTime,
			region			   String,
			location		   String,
			host			   String,
			method			   String,
			request_uri		   String,
			version			   String,
			category		   String,
			subject			   String,
			remote_addr		   FixedString(16),
			content_length	   Int64,
			os				   String,
			browser			   String,
			continent		   String,
			country			   String,
			iso_country		   String,
			city			   String,
			subdivision		   String,
			timezone		   String,
			duration_us		   UInt64,
			redis_duration_us  UInt64,
			db_duration_us	   UInt64,
			longitude		   Float64,
			latitude		   Float64,
			accuracy_radius	   UInt16,
			eu_member		   UInt8,
			width			   UInt32,
			height			   UInt32,
			user			   String,
			user_agent		   String,
			source			   Nullable(String),
			target			   Nullable(String),
			params			   Nullable(String),
			status			   UInt16,
			response		   Nullable(String),
			response_length	   UInt64,
			error			   Nullable(String),
			branch			   String,
			commit_hash		   FixedString(40),
			tag				   String,
			client_name		   String,
			client_branch	   String,
			client_commit_hash FixedString(40),
			client_tag		   String
		) engine=MergeTree() ORDER BY time PARTITION BY toYYYYMM(time)
	`)
	if err != nil {
		panic(err)
	}
	lr := AccessRecord{}
	l.availableFields = lr.GetAvailableFields()
	f := strings.Join(l.availableFields, ", ")
	placeholders := ":" + strings.Join(l.availableFields, ", :")
	queryTempl := "INSERT INTO %s (%s) VALUES (%s)"
	l.chInsertQuery = fmt.Sprintf(queryTempl, l.logTable, f, placeholders)
	records = make(chan AccessRecord, cs.MaxQueueSize)
	go l.send(cs)
}

func StopLogging() {
	close(records)
}

func (l *Logger) send(cs *ClickhouseSettings) {
	logStorage := make([]AccessRecord, 0, cs.MaxQueueSize)
	var ticker *time.Ticker
	if cs.Period.Seconds() != 0 {
		ticker = time.NewTicker(cs.Period)
	}
	for {
		log.Debug("Waiting message")
		select {
		case r, ok := <-records:
			if !ok {
				if cs.Connection.Debug {
					log.Debug("Channel is closed. Flushing")
				}
				err := l.flush(logStorage)
				if err != nil {
					log.Error(err)
					reserveRecords(logStorage, cs.Reserve)
				}
				logStorage = logStorage[:0]
			}
			logStorage = append(logStorage, r)
			if len(logStorage) > cs.BatchSize {
				if cs.Connection.Debug {
					log.Debug("Long queue. Flushing")
				}
				err := l.flush(logStorage)
				if err != nil {
					log.Error(err)
				} else {
					logStorage = logStorage[:0]
				}
			}
		case <-ticker.C:
			if cs.Connection.Debug {
				log.Debug("Time is up. Flushing")
			}
			err := l.flush(logStorage)
			if err != nil {
				log.Error(err)
			} else {
				logStorage = logStorage[:0]
			}
		}
	}
}

func (l *Logger) flush(logStorage []AccessRecord) error {
	localRecords := append(make([]AccessRecord, 0, len(logStorage)), logStorage...)
	tx, err := l.chWriter.Beginx()
	if err != nil {
		return err
	}
	nstmt, err := tx.PrepareNamed(l.chInsertQuery)
	if err != nil {
		return err
	}
	for _, r := range localRecords {
		_, err = nstmt.Exec(r)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}
