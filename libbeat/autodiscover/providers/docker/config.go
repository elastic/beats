package docker

import (
	"github.com/elastic/beats/libbeat/common/docker"
)

// Config for docker autodiscover provider
type Config struct {
	Host   string            `config:"host"`
	TLS    *docker.TLSConfig `config:"ssl"`
	Prefix string            `config:"string"`
}

func defaultConfig() *Config {
	return &Config{
		Host:   "unix:///var/run/docker.sock",
		Prefix: "co.elastic.",
	}
}

func (c *Config) Validate() {
	// Make sure that prefix ends with a '.'
	if c.Prefix[len(c.Prefix)-1] != '.' {
		c.Prefix = c.Prefix + "."
	}
}
