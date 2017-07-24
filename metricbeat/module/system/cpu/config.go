package cpu

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
)

// CPU metric types.
const (
	percentages           = "percentages"
	normalizedPercentages = "normalized_percentages"
	ticks                 = "ticks"
)

// Config for the system cpu metricset.
type Config struct {
	Metrics  []string `config:"cpu.metrics"`
	CPUTicks *bool    `config:"cpu_ticks"` // Deprecated.
}

// Validate validates the cpu config.
func (c Config) Validate() error {
	if c.CPUTicks != nil {
		cfgwarn.Deprecate("6.1", "cpu_ticks is deprecated. Add 'ticks' to the cpu.metrics list.")
	}

	if len(c.Metrics) == 0 {
		return errors.New("cpu.metrics cannot be empty")
	}

	for _, metric := range c.Metrics {
		switch strings.ToLower(metric) {
		case percentages, normalizedPercentages, ticks:
		default:
			return errors.Errorf("invalid cpu.metrics value '%v' (valid "+
				"options are %v, %v, and %v)", metric, percentages,
				normalizedPercentages, ticks)
		}
	}

	return nil
}

var defaultConfig = Config{
	Metrics: []string{percentages},
}
