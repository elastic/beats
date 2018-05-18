package jolokia

import (
	"time"

	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
)

// Config for Jolokia Discovery autodiscover provider
type Config struct {
	// List of network interfaces to use for discovery probes
	Interfaces []string

	// Time between discovery probes
	Period time.Duration `config:"duration"`

	// Time to wait till a response to a probe arrives
	ProbeTimeout time.Duration `config:"probe_timeout"`

	// Time since an instance is last seen and is considered removed
	GracePeriod time.Duration `config:"grace_period"`

	Builders  []*common.Config        `config:"builders"`
	Appenders []*common.Config        `config:"appenders"`
	Templates template.MapperSettings `config:"templates"`
}

func defaultConfig() *Config {
	return &Config{
		Period:       10 * time.Second,
		ProbeTimeout: 1 * time.Second,
		GracePeriod:  30 * time.Second,
		Interfaces:   []string{"any"},
	}
}
