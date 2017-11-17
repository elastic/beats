package core

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
)

// Core metric types.
const (
	percentages = "percentages"
	ticks       = "ticks"
)

// Config for the system core metricset.
type Config struct {
	Metrics  []string `config:"core.metrics"`
	CPUTicks *bool    `config:"cpu_ticks"` // Deprecated.
}

// Validate validates the core config.
func (c Config) Validate() error {
	if c.CPUTicks != nil {
		cfgwarn.Deprecate("6.1", "cpu_ticks is deprecated. Add 'ticks' to the core.metrics list.")
	}

	if len(c.Metrics) == 0 {
		return errors.New("core.metrics cannot be empty")
	}

	for _, metric := range c.Metrics {
		switch strings.ToLower(metric) {
		case percentages, ticks:
		default:
			return errors.Errorf("invalid core.metrics value '%v' (valid "+
				"options are %v and %v)", metric, percentages, ticks)
		}
	}

	return nil
}

var defaultConfig = Config{
	Metrics: []string{percentages},
}
