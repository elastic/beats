package process

import "github.com/elastic/beats/libbeat/logp"

// includeTopConfig is the configuration for the "top N processes
// filtering" feature
type includeTopConfig struct {
	Enabled  bool `config:"enabled"`
	ByCPU    int  `config:"by_cpu"`
	ByMemory int  `config:"by_memory"`
}

type Config struct {
	Procs           []string         `config:"processes"`
	Cgroups         *bool            `config:"process.cgroups.enabled"`
	EnvWhitelist    []string         `config:"process.env.whitelist"`
	CacheCmdLine    bool             `config:"process.cmdline.cache.enabled"`
	IncludeTop      includeTopConfig `config:"process.include_top_n"`
	IncludeCPUTicks bool             `config:"process.include_cpu_ticks"`
	CPUTicks        *bool            `config:"cpu_ticks"` // Deprecated
}

func (c Config) Validate() error {
	if c.CPUTicks != nil {
		logp.Deprecate("6.1", "cpu_ticks is deprecated. Use process.include_cpu_ticks instead")
	}
	return nil
}

var defaultConfig = Config{
	Procs:        []string{".*"}, // collect all processes by default
	CacheCmdLine: true,
	IncludeTop: includeTopConfig{
		Enabled:  true,
		ByCPU:    0,
		ByMemory: 0,
	},
}
