package beater

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
)

// Config is the root of the Metricbeat configuration hierarchy.
type Config struct {
	// Modules is a list of module specific configuration data.
	Modules       []*common.Config    `config:"modules"`
	ReloadModules ModulesReloadConfig `config:"reload.modules"`
}

type ModulesReloadConfig struct {
	// If path is a relative path, it is relative to the ${path.config}
	Path    string        `config:"path"`
	Period  time.Duration `config:"period"`
	Enabled bool          `config:"enabled"`
}

var (
	DefaultConfig = Config{
		ReloadModules: ModulesReloadConfig{
			Period:  10 * time.Second,
			Enabled: false,
		},
	}
)

func (c *ModulesReloadConfig) IsEnabled() bool {
	return c.Enabled
}
