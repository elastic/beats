package process

import (
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/metric/system/cgroup"
	"github.com/elastic/beats/v7/libbeat/opt"
)

type RunState byte

const (
	// RunStateSleep corresponds to a sleep state
	RunStateSleep = 'S'
	// RunStateRun corresponds to a running state
	RunStateRun = 'R'
	// RunStateStop corresponds to a stopped state
	RunStateStop = 'T'
	// RunStateZombie marks a zombie process
	RunStateZombie = 'Z'
	// RunStateIdle corresponds to an idle state
	RunStateIdle = 'D'
	// RunStateUnknown corresponds to a process in an unknown state
	RunStateUnknown = '?'
)

type ProcState struct {
	// Basic Process data
	Name     string  `struct:"name,omitempty"`
	State    string  `struct:"state,omitempty"`
	Username string  `struct:"username,omitempty"`
	pid      opt.Int `struct:"pid,omitempty"`
	Ppid     opt.Int `struct:"ppid,omitempty"`
	Pgid     opt.Int `struct:"pgid,omitempty"`

	// Extended Process Data
	Args    []string      `struct:"args,omitempty"`
	Cmdline opt.String    `struct:"cmdline,omitempty"`
	Cwd     opt.String    `struct:"cwd,omitempty"`
	Exe     opt.String    `struct:"exe,omitempty"`
	Env     common.MapStr `struct:"env,omitempty"`

	// Resource Metrics
	Memory ProcMemInfo `struct:"memory,omitempty"`
	CPU    ProcCPUInfo `struct:"cpu,omitempty"`
	FD     ProcFDInfo  `struct:"fd,omitempty"`

	// cgroups
	Cgroup cgroup.CGStats `struct:"cgroups,omitempty"`
}

type ProcCPUInfo struct {
	StartTime common.Time `struct:"start_time,omitempty"`
	Total     CPUTotal    `struct:"total,omitempty"`
	// Optional Tick values
	User   CPUTicks `struct:"user,omitempty"`
	System CPUTicks `struct:"system,omitempty"`
}

type CPUTicks struct {
	Ticks opt.Uint `struct:"ticks,omitempty"`
}

type CPUTotal struct {
	Value opt.Float  `struct:"value,omitempty"`
	Ticks opt.Uint   `struct:"ticks,omitempty"`
	Pct   opt.Float  `struct:"pct,omitempty"`
	Norm  opt.PctOpt `struct:"norm,omitempty"`
}

type ProcMemInfo struct {
	Size  opt.Uint   `struct:"size,omitempty"`
	Share opt.Uint   `struct:"share,omitempty"`
	Rss   MemBytePct `struct:"rss,omitempty"`
}

type MemBytePct struct {
	Bytes opt.Uint  `struct:"bytes,omitempty"`
	Pct   opt.Float `struct:"pct,omitempty"`
}

type ProcFDInfo struct {
	Open  opt.Uint   `struct:"open,omitempty"`
	Limit ProcLimits `struct:"limit,omitempty"`
}

type ProcLimits struct {
	Soft opt.Uint `struct:"soft,omitempty"`
	Hard opt.Uint `struct:"hard,omitempty"`
}
