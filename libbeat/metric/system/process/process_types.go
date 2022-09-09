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

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/metric/system/cgroup"
	"github.com/elastic/beats/v7/libbeat/opt"
)

// ProcState is the main struct for process information and metrics.
type ProcState struct {
	// Basic Process data
	Name     string   `struct:"name,omitempty"`
	State    PidState `struct:"state,omitempty"`
	Username string   `struct:"username,omitempty"`
	Pid      opt.Int  `struct:"pid,omitempty"`
	Ppid     opt.Int  `struct:"ppid,omitempty"`
	Pgid     opt.Int  `struct:"pgid,omitempty"`

	// Extended Process Data
	Args    []string      `struct:"args,omitempty"`
	Cmdline string        `struct:"cmdline,omitempty"`
	Cwd     string        `struct:"cwd,omitempty"`
	Exe     string        `struct:"exe,omitempty"`
	Env     common.MapStr `struct:"env,omitempty"`

	// Resource Metrics
	Memory ProcMemInfo `struct:"memory,omitempty"`
	CPU    ProcCPUInfo `struct:"cpu,omitempty"`
	FD     ProcFDInfo  `struct:"fd,omitempty"`

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

// IsZero returns true if the underlying value nil
func (t CPUTicks) IsZero() bool {
	return t.Ticks.IsZero()
}

// IsZero returns true if the underlying value nil
func (t ProcFDInfo) IsZero() bool {
	return t.Open.IsZero() && t.Limit.Hard.IsZero() && t.Limit.Soft.IsZero()
}

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
