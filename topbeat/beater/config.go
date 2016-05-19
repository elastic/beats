package beater

import (
	"errors"
	"time"
)

type ConfigSettings struct {
	Topbeat TopConfig `config:"topbeat"`
}

type TopConfig struct {
	Period time.Duration     `config:"period"  validate:"min=1ms"`
	Procs  []string          `config:"procs"`
	Stats  StatsEnableConfig `config:"stats"`
}

type StatsEnableConfig struct {
	System          bool `config:"system"`
	Proc            bool `config:"process"`
	Filesystem      bool `config:"filesystem"`
	CPUPerCore      bool `config:"cpu_per_core"`
	CPUTicks        bool `config:"cpu_ticks"`
	CPUTicksPerProc bool `config:"cpu_ticks_per_proc"`
}

var (
	defaultConfig = TopConfig{
		Period: 10 * time.Second,
		Procs:  []string{".*"}, //all processes
		Stats: StatsEnableConfig{
			System:          true,
			Proc:            true,
			Filesystem:      true,
			CPUPerCore:      false,
			CPUTicks:        false,
			CPUTicksPerProc: false,
		},
	}
)

func (c *TopConfig) Validate() error {
	return nil
}

func (c *StatsEnableConfig) Validate() error {
	if !c.System && !c.Proc && !c.Filesystem {
		return errors.New("Invalid statistics configuration (enable one of 'system', 'process' or 'filesystem')")
	}

	return nil
}
