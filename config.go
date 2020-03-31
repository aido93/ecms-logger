package ECMSLogger

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"time"
)

type (
	MaxMind struct {
		DB     string            `yaml:"db"`
		Source map[string]string `yaml:"source"`
	}

	RotateConf struct {
		MaxFiles int    `yaml:"maxFiles"`
		MaxSize  string `yaml:"maxSize"`
	}

	Reserve struct {
		Dir    string     `yaml:"dir"`
		Rotate RotateConf `yaml:"rotate"`
	}

	Connection struct {
		Host      string        `yaml:"host"`
		Port      string        `yaml:"port"`
		User      string        `yaml:"user"`
		Password  string        `yaml:"password"`
		DB        string        `yaml:"db"`
		AltHosts  []string      `yaml:"altHosts"`
		ConnLimit int           `yaml:"connLimit"`
		IdleLimit int           `yaml:"idleLimit"`
		Timeout   time.Duration `yaml:"timeout"`
		Debug     bool          `yaml:"debug"`
	}

	ClickhouseSettings struct {
		Connection   Connection    `yaml:"connection"`
		Table        string        `yaml:"table"`
		BatchSize    int           `yaml:"batchSize"`
		BlockSize    int           `yaml:"blockSize"`
		PoolSize     int           `yaml:"poolSize"`
		MaxQueueSize int           `yaml:"maxQueueSize"`
		Period       time.Duration `yaml:"period"`
		Reserve      *Reserve      `yaml:"reserve"`
	}

	Fields struct {
		String *[]string `yaml:"string"`
		Bool   *[]string `yaml:"bool"`
		Int    *[]string `yaml:"int"`
	}

	Session struct {
		Cookie         string        `yaml:"cookie"`
		MaxAge         time.Duration `yaml:"maxAge"`
		Secure         bool          `yaml:"secure"`
		OptionalFields []string      `yaml:"optionalFields"`
		Fields         Fields        `yaml:"fields"`
	}

	Config struct {
		MaxMind    MaxMind            `yaml:"maxmind"`
		Clickhouse ClickhouseSettings `yaml:"clickhouse"`
		Session    Session            `yaml:"session"`
	}
)

func ReadConfig(filename string) (Config, error) {
	c := Config{}
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}
	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		return Config{}, err
	}
	return c, nil
}
