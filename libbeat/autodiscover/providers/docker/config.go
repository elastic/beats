package docker

import (
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/docker"
)

// Config for docker autodiscover provider
type Config struct {
	Host      string                  `config:"host"`
	TLS       *docker.TLSConfig       `config:"ssl"`
	Prefix    string                  `config:"string"`
	Builders  []*common.Config        `config:"builders"`
	Templates template.MapperSettings `config:"templates"`
}

func defaultConfig() *Config {
	return &Config{
		Host:   "unix:///var/run/docker.sock",
		Prefix: "co.elastic.",
	}
}

// Validate ensures correctness of config
func (c *Config) Validate() {
	// Make sure that prefix ends with a '.'
	if c.Prefix[len(c.Prefix)-1] != '.' {
		c.Prefix = c.Prefix + "."
	}
}
