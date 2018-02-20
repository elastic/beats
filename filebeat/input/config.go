package input

import (
	"time"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
)

var (
	defaultConfig = inputConfig{
		ScanFrequency: 10 * time.Second,
		Type:          cfg.DefaultType,
	}
)

type inputConfig struct {
	ScanFrequency time.Duration `config:"scan_frequency" validate:"min=0,nonzero"`
	Type          string        `config:"type"`
	InputType     string        `config:"input_type"`
}

func (c *inputConfig) Validate() error {
	if c.InputType != "" {
		cfgwarn.Deprecate("6.0.0", "input_type input config is deprecated. Use type instead.")
		c.Type = c.InputType
	}
	return nil
}
