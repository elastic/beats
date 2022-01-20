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
	Pid      opt.Int `struct:"pid,omitempty"`
	Ppid     opt.Int `struct:"ppid,omitempty"`
	Pgid     opt.Int `struct:"pgid,omitempty"`

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
	Cgroup cgroup.CGStats `struct:"cgroups,omitempty"`

	// meta
	SampleTime time.Time `struct:"-,omitempty"`
}

type ProcCPUInfo struct {
	StartTime string   `struct:"start_time,omitempty"`
	Total     CPUTotal `struct:"total,omitempty"`
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

// Implementations

// IsZero returns true if the underlying value nil
func (t CPUTicks) IsZero() bool {
	return t.Ticks.IsZero()
}
