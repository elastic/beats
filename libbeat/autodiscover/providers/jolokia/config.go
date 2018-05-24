package jolokia

import (
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
)

var (
	defaultInterval     = 10 * time.Second
	defaultProbeTimeout = 1 * time.Second
	defaultGracePeriod  = 30 * time.Second
)

// Config for Jolokia Discovery autodiscover provider
type Config struct {
	// List of network interfaces to use for discovery probes
	Interfaces []InterfaceConfig

	Builders  []*common.Config        `config:"builders"`
	Appenders []*common.Config        `config:"appenders"`
	Templates template.MapperSettings `config:"templates"`
}

// InterfaceConfig is the configuration for a network interface used for probes
type InterfaceConfig struct {
	// Name of the interface
	Name string `config:"name"`

	// Time between discovery probes
	Interval time.Duration `config:"interval"`

	// Time to wait till a response to a probe arrives
	ProbeTimeout time.Duration `config:"probe_timeout"`

	// Time since an instance is last seen and is considered removed
	GracePeriod time.Duration `config:"grace_period"`
}

// WithDefaults returns an InterfaceConfig with default values
func (c InterfaceConfig) WithDefaults() InterfaceConfig {
	if c.Interval == 0 {
		c.Interval = defaultInterval
	}
	if c.ProbeTimeout == 0 {
		c.ProbeTimeout = defaultProbeTimeout
	}
	if c.GracePeriod == 0 {
		c.GracePeriod = defaultGracePeriod
	}

	// Avoid having sockets open more time than needed
	if c.ProbeTimeout > c.Interval {
		c.ProbeTimeout = c.Interval
	}

	return c
}

func defaultConfig() *Config {
	return &Config{}
}

func getConfig(c *common.Config) (*Config, error) {
	config := defaultConfig()
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	for i, iface := range config.Interfaces {
		if len(iface.Name) == 0 {
			return nil, errors.New("interface without name")
		}
		config.Interfaces[i] = iface.WithDefaults()
	}

	if len(config.Interfaces) == 0 {
		config.Interfaces = []InterfaceConfig{
			InterfaceConfig{Name: "any"}.WithDefaults(),
		}
	}

	return config, nil
}
