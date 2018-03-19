package beater

import (
	"time"

	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/common"
)

// Config is the root of the Metricbeat configuration hierarchy.
type Config struct {
	// Modules is a list of module specific configuration data.
	Modules       []*common.Config     `config:"modules"`
	ConfigModules *common.Config       `config:"config.modules"`
	MaxStartDelay time.Duration        `config:"max_start_delay"` // Upper bound on the random startup delay for metricsets (use 0 to disable startup delay).
	Autodiscover  *autodiscover.Config `config:"autodiscover"`
}

var defaultConfig = Config{
	MaxStartDelay: 10 * time.Second,
}
