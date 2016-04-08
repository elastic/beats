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
	LoadBalance bool                  `config:"loadbalance"`
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
		Port:             6379,
		LoadBalance:      true,
		Timeout:          5 * time.Second,
		MaxRetries:       3,
		TLS:              nil,
		Db:               0,
		DataType:         "list",
		HostTopology:     "",
		PasswordTopology: "",
		DbTopology:       1,
	}
)
