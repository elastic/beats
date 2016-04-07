package redis

import (
	"time"

	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

type redisConfig struct {
	Password    string                `config:"password"`
	Index       string                `config:"index"`
	Port        int                   `config:"port"`
	LoadBalance bool                  `config:"LoadBalance"`
	Timeout     time.Duration         `config:"timeout"`
	MaxRetries  int                   `config:"max_retries"`
	TLS         *outputs.TLSConfig    `config:"tls"`
	Proxy       transport.ProxyConfig `config:",inline"`

	Db       int    `config:"db"`
	DataType string `config:"datatype"`

	HostTopology     string `config:"host_topology"`
	PasswordTopology string `config:"password_topology"`
	DbTopology       int    `config:"db_topology"`
}

var (
	defaultConfig = redisConfig{
		Port:        6379,
		LoadBalance: true,
		Timeout:     30 * time.Second,
		MaxRetries:  3,
	}
)

/*
type redisConfig struct {
	Host              string        `config:"host"`
	Port              int           `config:"port"`
	Password          string        `config:"password"`
	Db                int           `config:"db"`
	DbTopology        int           `config:"db_topology"`
	Timeout           time.Duration `config:"timeout"`
	Index             string        `config:"index"`
	ReconnectInterval int           `config:"reconnect_interval"`
	DataType          string        `config:"datatype"`
}

var (
	defaultConfig = redisConfig{
		DbTopology:        1,
		Timeout:           5 * time.Second,
		ReconnectInterval: 1,
	}
)
*/
