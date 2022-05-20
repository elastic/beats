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

//go:build darwin || freebsd || linux || windows || aix || netbsd || openbsd
// +build darwin freebsd linux windows aix netbsd openbsd

package process

import (
	"errors"
	"fmt"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/match"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/cgroup"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
	"github.com/elastic/go-sysinfo/types"

	sysinfo "github.com/elastic/go-sysinfo"
)

// ProcsMap is a map where the keys are the names of processes and the value is the Process with that name
type ProcsMap map[int]ProcState

// ProcCallback is a function that FetchPid* methods can call at various points to do OS-agnostic processing
type ProcCallback func(in ProcState) (ProcState, error)

// CgroupPctStats stores rendered percent values from cgroup CPU data
type CgroupPctStats struct {
	CPUTotalPct      float64
	CPUTotalPctNorm  float64
	CPUUserPct       float64
	CPUUserPctNorm   float64
	CPUSystemPct     float64
	CPUSystemPctNorm float64
}

// Stats stores the stats of processes on the host.
type Stats struct {
	Hostfs        resolve.Resolver
	Procs         []string
	ProcsMap      ProcsMap
	CPUTicks      bool
	EnvWhitelist  []string
	CacheCmdLine  bool
	IncludeTop    IncludeTopConfig
	CgroupOpts    cgroup.ReaderOptions
	EnableCgroups bool

	skipExtended bool
	procRegexps  []match.Matcher // List of regular expressions used to whitelist processes.
	envRegexps   []match.Matcher // List of regular expressions used to whitelist env vars.
	cgroups      *cgroup.Reader
	logger       *logp.Logger
	host         types.Host
}

//PidState are the constants for various PID states
type PidState string

var (
	//Dead state, on linux this is both "x" and "X"
	Dead PidState = "dead"
	//Running state
	Running PidState = "running"
	//Sleeping state
	Sleeping PidState = "sleeping"
	//Idle state. On linux this is "D"
	Idle PidState = "idle"
	//Stopped state.
	Stopped PidState = "stopped"
	//Zombie state.
	Zombie PidState = "zombie"
	//WakeKill is a linux state only found on kernels 2.6.33-3.13
	WakeKill PidState = "wakekill"
	//Waking  is a linux state only found on kernels 2.6.33-3.13
	Waking PidState = "waking"
	//Parked is a linux state. On the proc man page, it says it's available on 3.9-3.13, but it appears to still be in the code.
	Parked PidState = "parked"
	//Unknown state
	Unknown PidState = "unknown"
)

// PidStates is a Map of all pid states, mostly applicable to linux
var PidStates = map[byte]PidState{
	'S': Sleeping,
	'R': Running,
	'D': Idle, // Waiting in uninterruptible disk sleep, on some kernels this is marked as I below
	'I': Idle, // in the scheduler, TASK_IDLE is defined as (TASK_UNINTERRUPTIBLE | TASK_NOLOAD)
	'T': Stopped,
	'Z': Zombie,
	'X': Dead,
	'x': Dead,
	'K': WakeKill,
	'W': Waking,
	'P': Parked,
}

// Init initializes a Stats instance. It returns errors if the provided process regexes
// cannot be compiled.
func (procStats *Stats) Init() error {
	procStats.logger = logp.NewLogger("processes")
	var err error
	procStats.host, err = sysinfo.Host()
	if err != nil {
		procStats.host = nil
		procStats.logger.Warnf("Getting host details: %v", err)
	}

	//footcannon prevention
	if procStats.Hostfs == nil {
		procStats.Hostfs = resolve.NewTestResolver("/")
	}

	procStats.ProcsMap = make(ProcsMap)

	if len(procStats.Procs) == 0 {
		return nil
	}

	procStats.procRegexps = []match.Matcher{}
	for _, pattern := range procStats.Procs {
		reg, err := match.Compile(pattern)
		if err != nil {
			return fmt.Errorf("failed to compile regexp [%s]: %w", pattern, err)
		}
		procStats.procRegexps = append(procStats.procRegexps, reg)
	}

	procStats.envRegexps = make([]match.Matcher, 0, len(procStats.EnvWhitelist))
	for _, pattern := range procStats.EnvWhitelist {
		reg, err := match.Compile(pattern)
		if err != nil {
			return fmt.Errorf("failed to compile env whitelist regexp [%v]: %w", pattern, err)
		}
		procStats.envRegexps = append(procStats.envRegexps, reg)
	}

	if procStats.EnableCgroups {
		cgReader, err := cgroup.NewReaderOptions(procStats.CgroupOpts)
		if errors.Is(err, cgroup.ErrCgroupsMissing) {
			logp.Warn("cgroup data collection will be disabled: %v", err)
			procStats.EnableCgroups = false
		} else if err != nil {
			return fmt.Errorf("error initializing cgroup reader: %w", err)
		}
		procStats.cgroups = cgReader
	}
	return nil
}
