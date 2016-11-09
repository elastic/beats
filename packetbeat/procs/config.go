package procs

import "time"

type ProcsConfig struct {
	Enabled         bool          `config:"enabled"`
	MaxProcReadFreq time.Duration `config:"max_proc_read_freq"`
	Monitored       []procConfig  `config:"monitored"`
	RefreshPidsFreq time.Duration `config:"refresh_pids_freq"`
}
