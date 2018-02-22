package process

import (
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/metric/system/process"
)

type Config struct {
	Procs           []string                 `config:"processes"`
	Cgroups         *bool                    `config:"process.cgroups.enabled"`
	EnvWhitelist    []string                 `config:"process.env.whitelist"`
	CacheCmdLine    bool                     `config:"process.cmdline.cache.enabled"`
	IncludeTop      process.IncludeTopConfig `config:"process.include_top_n"`
	IncludeCPUTicks bool                     `config:"process.include_cpu_ticks"`
	CPUTicks        *bool                    `config:"cpu_ticks"` // Deprecated
}

func (c Config) Validate() error {
	if c.CPUTicks != nil {
		cfgwarn.Deprecate("6.1", "cpu_ticks is deprecated. Use process.include_cpu_ticks instead")
	}
	return nil
}

var defaultConfig = Config{
	Procs:        []string{".*"}, // collect all processes by default
	CacheCmdLine: true,
	IncludeTop: process.IncludeTopConfig{
		Enabled:  true,
		ByCPU:    0,
		ByMemory: 0,
	},
}
