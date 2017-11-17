package docker

import (
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common/docker"
)

// Config for docker autodiscover provider
type Config struct {
	Host      string                  `config:"host"`
	TLS       *docker.TLSConfig       `config:"ssl"`
	Templates template.MapperSettings `config:"templates"`
}

func defaultConfig() *Config {
	return &Config{
		Host: "unix:///var/run/docker.sock",
	}
}
