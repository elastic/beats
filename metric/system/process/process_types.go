// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package process

import (
	"time"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/cgroup"
	sysinfotypes "github.com/elastic/go-sysinfo/types"
)

// ProcState is the main struct for process information and metrics.
type ProcState struct {
	// Basic Process data
	Name       string   `struct:"name,omitempty"`
	State      PidState `struct:"state,omitempty"`
	Username   string   `struct:"username,omitempty"`
	Pid        opt.Int  `struct:"pid,omitempty"`
	Ppid       opt.Int  `struct:"ppid,omitempty"`
	Pgid       opt.Int  `struct:"pgid,omitempty"`
	NumThreads opt.Int  `struct:"num_threads,omitempty"`

	// Extended Process Data
	Args    []string `struct:"args,omitempty"`
	Cmdline string   `struct:"cmdline,omitempty"`
	Cwd     string   `struct:"cwd,omitempty"`
	Exe     string   `struct:"exe,omitempty"`
	Env     mapstr.M `struct:"env,omitempty"`

	// Resource Metrics
	Memory  ProcMemInfo                       `struct:"memory,omitempty"`
	CPU     ProcCPUInfo                       `struct:"cpu,omitempty"`
	FD      ProcFDInfo                        `struct:"fd,omitempty"`
	Network *sysinfotypes.NetworkCountersInfo `struct:"-,omitempty"`
	IO      ProcIOInfo                        `struct:"io,omitempty"`

	// cgroups
	Cgroup cgroup.CGStats `struct:"cgroup,omitempty"`

	// meta
	SampleTime time.Time `struct:"-,omitempty"`
}

// ProcCPUInfo is the main struct for CPU metrics
type ProcCPUInfo struct {
	StartTime string   `struct:"start_time,omitempty"`
	Total     CPUTotal `struct:"total,omitempty"`
	// Optional Tick values
	User   CPUTicks `struct:"user,omitempty"`
	System CPUTicks `struct:"system,omitempty"`
}

// CPUTicks is a formatting wrapper for `tick` metric values
type CPUTicks struct {
	Ticks opt.Uint `struct:"ticks,omitempty"`
}

// CPUTotal is the struct for cpu.total metrics
type CPUTotal struct {
	Value opt.Float  `struct:"value,omitempty"`
	Ticks opt.Uint   `struct:"ticks,omitempty"`
	Pct   opt.Float  `struct:"pct,omitempty"`
	Norm  opt.PctOpt `struct:"norm,omitempty"`
}

// ProcIOInfo is the struct for I/O counters from /proc/[pid]/io
type ProcIOInfo struct {
	// ReadChar is bytes read from the system, as passed from read() and similar syscalls
	ReadChar opt.Uint `struct:"read_char,omitempty"`
	// WriteChar is bytes written to the system, as passed to various syscalls
	WriteChar opt.Uint `struct:"write_char,omitempty"`
	//ReadSyscalls counts the number of read operations
	ReadSyscalls opt.Uint `struct:"read_ops,omitempty"`
	//WriteSyscalls counts the number of write operations
	WriteSyscalls opt.Uint `struct:"write_ops,omitempty"`
	// ReadBytes is the count of bytes that were actually fetched from the storage layer
	ReadBytes opt.Uint `struct:"read_bytes,omitempty"`
	// WriteBytes is the count of bytes that were actually written to the storage layer
	WriteBytes opt.Uint `struct:"write_bytes,omitempty"`
	// the number of bytes which this process caused to not happen, by truncating pagecache
	CancelledWriteBytes opt.Uint `struct:"cancelled_write_bytes,omitempty"`
}

// ProcMemInfo is the struct for cpu.memory metrics
type ProcMemInfo struct {
	Size  opt.Uint   `struct:"size,omitempty"`
	Share opt.Uint   `struct:"share,omitempty"`
	Rss   MemBytePct `struct:"rss,omitempty"`
}

// MemBytePct is the formatting struct for wrapping pct/byte metrics
type MemBytePct struct {
	Bytes opt.Uint  `struct:"bytes,omitempty"`
	Pct   opt.Float `struct:"pct,omitempty"`
}

// ProcFDInfo is the struct for process.fd metrics
type ProcFDInfo struct {
	Open  opt.Uint   `struct:"open,omitempty"`
	Limit ProcLimits `struct:"limit,omitempty"`
}

// ProcLimits wraps the fd.limit metrics
type ProcLimits struct {
	Soft opt.Uint `struct:"soft,omitempty"`
	Hard opt.Uint `struct:"hard,omitempty"`
}

// Implementations

func (t CPUTotal) IsZero() bool {
	return t.Value.IsZero() && t.Ticks.IsZero() && t.Pct.IsZero() && t.Norm.IsZero()
}

// IsZero returns true if the underlying value nil
func (t CPUTicks) IsZero() bool {
	return t.Ticks.IsZero()
}

// IsZero returns true if the underlying value nil
func (t ProcFDInfo) IsZero() bool {
	return t.Open.IsZero() && t.Limit.Hard.IsZero() && t.Limit.Soft.IsZero()
}

// FormatForRoot takes the ProcState event and turns the fields into a ProcStateRootEvent
// struct. These are the root fields expected for events sent from the system/process metricset.
func (p *ProcState) FormatForRoot() ProcStateRootEvent {
	root := ProcStateRootEvent{}

	root.Process.Name = p.Name
	p.Name = ""

	root.Process.Pid = p.Pid
	p.Pid = opt.NewIntNone()

	root.Process.Parent.Pid = p.Ppid
	p.Ppid = opt.NewIntNone()

	root.Process.Pgid = p.Pgid
	p.Pgid = opt.NewIntNone()

	root.User.Name = p.Username
	p.Username = ""

	root.Process.Cmdline = p.Cmdline
	root.Process.State = p.State
	root.Process.CPU.StartTime = p.CPU.StartTime
	root.Process.CPU.Pct = p.CPU.Total.Norm.Pct
	root.Process.Memory.Pct = p.Memory.Rss.Pct

	root.Process.Cwd = p.Cwd
	p.Cwd = ""

	root.Process.Exe = p.Exe
	p.Exe = ""

	root.Process.Args = p.Args
	p.Args = nil

	return root
}

// ProcStateRootEvent represents the "root" beat/agent ECS event fields that are copied from the integration-level event.
type ProcStateRootEvent struct {
	Process ProcessRoot `struct:"process,omitempty"`
	User    Name        `struct:"user,omitempty"`
}

// ProcessRoot wraps the process metrics for the root ECS fields
type ProcessRoot struct {
	Cmdline string        `struct:"command_line,omitempty"`
	State   PidState      `struct:"state,omitempty"`
	CPU     RootCPUFields `struct:"cpu,omitempty"`
	Memory  opt.PctOpt    `struct:"memory,omitempty"`
	Cwd     string        `struct:"working_directory,omitempty"`
	Exe     string        `struct:"executable,omitempty"`
	Args    []string      `struct:"args,omitempty"`
	Name    string        `struct:"name,omitempty"`
	Pid     opt.Int       `struct:"pid,omitempty"`
	Parent  Parent        `struct:"parent,omitempty"`
	Pgid    opt.Int       `struct:"pgid,omitempty"`
}

type Parent struct {
	Pid opt.Int `struct:"pid,omitempty"`
}

type Name struct {
	Name string `struct:"name,omitempty"`
}

type RootCPUFields struct {
	StartTime string    `struct:"start_time,omitempty"`
	Pct       opt.Float `struct:"pct,omitempty"`
}
