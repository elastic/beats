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

//go:build darwin || freebsd || linux || windows || aix
// +build darwin freebsd linux windows aix

package process

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/go-sysinfo/types"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/metric/system/cgroup"
	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/libbeat/opt"
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

// ListStates is a wrapper that returns a list of processess with only the basic PID info filled out.
func ListStates(hostfs resolve.Resolver) ([]ProcState, error) {
	init := Stats{
		Hostfs:        hostfs,
		Procs:         []string{".*"},
		EnableCgroups: false,
		skipExtended:  true,
	}
	err := init.Init()
	if err != nil {
		return nil, errors.Wrap(err, "error initializing process collectors")
	}

	// actually fetch the PIDs from the OS-specific code
	_, plist, err := init.FetchPids()
	if err != nil {
		return nil, errors.Wrap(err, "error gathering PIDs")
	}

	return plist, nil
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
			return fmt.Errorf("Failed to compile regexp [%s]: %v", pattern, err)
		}
		procStats.procRegexps = append(procStats.procRegexps, reg)
	}

	procStats.envRegexps = make([]match.Matcher, 0, len(procStats.EnvWhitelist))
	for _, pattern := range procStats.EnvWhitelist {
		reg, err := match.Compile(pattern)
		if err != nil {
			return fmt.Errorf("failed to compile env whitelist regexp [%v]: %v", pattern, err)
		}
		procStats.envRegexps = append(procStats.envRegexps, reg)
	}

	if procStats.EnableCgroups {
		cgReader, err := cgroup.NewReaderOptions(procStats.CgroupOpts)
		if err == cgroup.ErrCgroupsMissing {
			logp.Warn("cgroup data collection will be disabled: %v", err)
		} else if err != nil {
			return errors.Wrap(err, "error initializing cgroup reader")
		}
		procStats.cgroups = cgReader
	}
	return nil
}

// Get fetches the configured processes and returns a list of formatted events and root ECS fields
func (procStats *Stats) Get() ([]common.MapStr, []common.MapStr, error) {
	//If the user hasn't configured any kind of process glob, return
	if len(procStats.Procs) == 0 {
		return nil, nil, nil
	}

	// actually fetch the PIDs from the OS-specific code
	pidMap, plist, err := procStats.FetchPids()

	if err != nil {
		return nil, nil, errors.Wrap(err, "error gathering PIDs")
	}
	// We use this to track processes over time.
	procStats.ProcsMap = pidMap

	// filter the process list that will be passed down to users
	procStats.includeTopProcesses(plist)

	// This is a holdover until we migrate this library to metricbeat/internal
	// At which point we'll use the memory code there.
	var totalPhyMem uint64
	if procStats.host != nil {
		memStats, err := procStats.host.Memory()
		if err != nil {
			procStats.logger.Warnf("Getting memory details: %v", err)
		} else {
			totalPhyMem = memStats.Total
		}

	}

	//Format the list to the MapStr type used by the outputs
	procs := []common.MapStr{}
	rootEvents := []common.MapStr{}

	for _, process := range plist {
		// Add the RSS pct memory first
		process.Memory.Rss.Pct = GetProcMemPercentage(process, totalPhyMem)
		//Create the root event
		root := process.FormatForRoot()
		rootMap := common.MapStr{}
		err := typeconv.Convert(&rootMap, root)

		proc, err := procStats.getProcessEvent(&process)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "error converting process for pid %d", process.Pid.ValueOr(0))
		}

		procs = append(procs, proc)
		rootEvents = append(rootEvents, rootMap)
	}

	return procs, rootEvents, nil
}

// GetOne fetches process data for a given PID if its name matches the regexes provided from the host.
func (procStats *Stats) GetOne(pid int) (common.MapStr, error) {
	pidStat, _, err := procStats.pidFill(pid, false)
	if err != nil {
		return nil, errors.Wrapf(err, "error fetching PID %d", pid)
	}
	newMap := make(ProcsMap)
	newMap[pid] = pidStat
	procStats.ProcsMap = newMap

	return procStats.getProcessEvent(&pidStat)
}

// GetSelf gets process info for the beat itself
func (procStats *Stats) GetSelf() (ProcState, error) {
	self := os.Getpid()

	pidStat, _, err := procStats.pidFill(self, false)
	if err != nil {
		return ProcState{}, errors.Wrapf(err, "error fetching PID %d", self)
	}

	return pidStat, nil
}

// pidIter wraps a few lines of generic code that all OS-specific FetchPids() functions must call.
// this also handles the process of adding to the maps/lists in order to limit the code duplication in all the OS implementations
func (procStats *Stats) pidIter(pid int, procMap map[int]ProcState, proclist []ProcState) (map[int]ProcState, []ProcState) {
	status, saved, err := procStats.pidFill(pid, true)
	if err != nil {
		procStats.logger.Debugf("Error fetching PID info for %d, skipping: %s", pid, err)
		return procMap, proclist
	}
	if !saved {
		procStats.logger.Debugf("Process name does not match the provided regex; PID=%d; name=%s", pid, status.Name)
		return procMap, proclist
	}
	procMap[pid] = status
	proclist = append(proclist, status)

	return procMap, proclist
}

// pidFill is an entrypoint used by OS-specific code to fill out a pid.
// This in turn calls various OS-specific code to fill out the various bits of PID data
// This is done to minimize the code duplication between different OS implementations
// The second return value will only be false if an event has been filtered out
func (procStats *Stats) pidFill(pid int, filter bool) (ProcState, bool, error) {
	// Fetch proc state so we can get the name for filtering based on user's filter.

	// OS-specific entrypoint, get basic info so we can at least run matchProcess
	status, err := GetInfoForPid(procStats.Hostfs, pid)
	if err != nil {
		return status, true, errors.Wrap(err, "GetInfoForPid")
	}
	if procStats.skipExtended {
		return status, true, nil
	}
	status = procStats.cacheCmdLine(status)

	// Filter based on user-supplied func
	if filter {
		if !procStats.matchProcess(status.Name) {
			return status, false, nil
		}
	}

	//If we've passed the filter, continue to fill out the rest of the metrics
	status, err = FillPidMetrics(procStats.Hostfs, pid, status, procStats.isWhitelistedEnvVar)
	if err != nil {
		return status, true, errors.Wrap(err, "FillPidMetrics")
	}
	if len(status.Args) > 0 && status.Cmdline == "" {
		status.Cmdline = strings.Join(status.Args, " ")
	}

	//postprocess with cgroups and percentages
	last, ok := procStats.ProcsMap[status.Pid.ValueOr(0)]
	status.SampleTime = time.Now()
	if procStats.EnableCgroups {
		cgStats, err := procStats.cgroups.GetStatsForPid(status.Pid.ValueOr(0))
		if err != nil {
			return status, true, errors.Wrap(err, "cgroups.GetStatsForPid")
		}
		status.Cgroup = cgStats
		if ok {
			status.Cgroup.FillPercentages(last.Cgroup, status.SampleTime, last.SampleTime)
		}
	} // end cgroups processor

	if ok {
		status = GetProcCPUPercentage(last, status)
	}

	return status, true, nil
}

// cacheCmdLine fills out Env and arg metrics from any stored previous metrics for the pid
func (procStats *Stats) cacheCmdLine(in ProcState) ProcState {
	if previousProc, ok := procStats.ProcsMap[in.Pid.ValueOr(0)]; ok {
		if procStats.CacheCmdLine {
			in.Args = previousProc.Args
			in.Cmdline = previousProc.Cmdline
		}
		env := previousProc.Env
		in.Env = env
	}
	return in
}

// return a formatted MapStr of the process metrics
func (procStats *Stats) getProcessEvent(process *ProcState) (common.MapStr, error) {

	// Remove CPUTicks if needed
	if !procStats.CPUTicks {
		process.CPU.User.Ticks = opt.NewUintNone()
		process.CPU.System.Ticks = opt.NewUintNone()
		process.CPU.Total.Ticks = opt.NewUintNone()
	}

	proc := common.MapStr{}
	err := typeconv.Convert(&proc, process)

	return proc, err
}

// matchProcess checks if the provided process name matches any of the process regexes
func (procStats *Stats) matchProcess(name string) bool {
	for _, reg := range procStats.procRegexps {
		if reg.MatchString(name) {
			return true
		}
	}
	return false
}

// includeTopProcesses filters down the metrics based on top CPU or top Memory settings
func (procStats *Stats) includeTopProcesses(processes []ProcState) []ProcState {
	if !procStats.IncludeTop.Enabled ||
		(procStats.IncludeTop.ByCPU == 0 && procStats.IncludeTop.ByMemory == 0) {

		return processes
	}

	var result []ProcState
	if procStats.IncludeTop.ByCPU > 0 {
		numProcs := procStats.IncludeTop.ByCPU
		if len(processes) < procStats.IncludeTop.ByCPU {
			numProcs = len(processes)
		}

		sort.Slice(processes, func(i, j int) bool {
			return processes[i].CPU.Total.Pct.ValueOr(0) > processes[j].CPU.Total.Pct.ValueOr(0)
		})
		result = append(result, processes[:numProcs]...)
	}

	if procStats.IncludeTop.ByMemory > 0 {
		numProcs := procStats.IncludeTop.ByMemory
		if len(processes) < procStats.IncludeTop.ByMemory {
			numProcs = len(processes)
		}

		sort.Slice(processes, func(i, j int) bool {
			return processes[i].Memory.Rss.Bytes.ValueOr(0) > processes[j].Memory.Rss.Bytes.ValueOr(0)
		})
		for _, proc := range processes[:numProcs] {
			if !isProcessInSlice(result, &proc) {
				result = append(result, proc)
			}
		}
	}

	return result
}

// isWhitelistedEnvVar returns true if the given variable name is a match for
// the whitelist. If the whitelist is empty it returns false.
func (procStats Stats) isWhitelistedEnvVar(varName string) bool {
	if len(procStats.envRegexps) == 0 {
		return false
	}

	for _, p := range procStats.envRegexps {
		if p.MatchString(varName) {
			return true
		}
	}
	return false
}
