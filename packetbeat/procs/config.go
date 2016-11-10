package procs

import "time"

type ProcsConfig struct {
	Enabled         bool          `config:"enabled"`
	MaxProcReadFreq time.Duration `config:"max_proc_read_freq"`
	Monitored       []ProcConfig  `config:"monitored"`
	RefreshPidsFreq time.Duration `config:"refresh_pids_freq"`
}

type ProcConfig struct {
	Process     string `config:"process"`
	CmdlineGrep string `config:"cmdline_grep"`
}
