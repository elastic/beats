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
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/go-sysinfo/types"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/metric/system/cgroup"
	"github.com/elastic/beats/v7/libbeat/metric/system/numcpu"
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

	procRegexps []match.Matcher // List of regular expressions used to whitelist processes.
	envRegexps  []match.Matcher // List of regular expressions used to whitelist env vars.
	cgroups     *cgroup.Reader
	logger      *logp.Logger
	host        types.Host
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

// Get fetches the configured processes and returns a formatted map
func (procStats *Stats) Get() ([]common.MapStr, error) {
	//If the user hasn't configured any kind of process glob, return
	if len(procStats.Procs) == 0 {
		return nil, nil
	}

	// actually fetch the PIDs from the OS-specific code
	pidMap, plist, err := procStats.FetchPids()

	if err != nil {
		return nil, errors.Wrap(err, "error gathering PIDs")
	}
	// We use this to track processes over time.
	procStats.ProcsMap = pidMap

	// filter the process list that will be passed down to users
	procStats.includeTopProcesses(plist)

	procs := make([]common.MapStr, 0, len(plist))
	for _, process := range plist {
		proc, err := procStats.getProcessEvent(&process)
		if err != nil {
			return nil, errors.Wrapf(err, "error converting process for pid %d", process.Pid.ValueOr(0))
		}
		procs = append(procs, proc)
	}

	return procs, nil
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

// pidFill is an entrypoint used by OS-specific code to fill out a pid.
// This in turn calls various OS-specific code to fill out the various bits of PID data
// This is done to minimize the code duplication between different OS implementations
// The second return value will only be false if an event has been filtered out
func (procStats *Stats) pidFill(pid int, filter bool) (ProcState, bool, error) {
	// Fetch proc state so we can get the name for filtering based on user's filter.

	status, err := GetInfoForPid(procStats.Hostfs, pid)
	if err != nil {
		return status, true, errors.Wrap(err, "GetInfoForPid")
	}
	status = procStats.cacheCmdLine(status)
	// Filter based on user-supplied func
	if filter {
		if !procStats.matchProcess(status.Name) {
			return status, false, nil
		}
	}

	//If we've passed the filter, continue to fill out the rest of the metrics
	status, err = FillPidMetrics(procStats.Hostfs, pid, status)
	if err != nil {
		return status, true, errors.Wrap(err, "FillPidMetrics")
	}

	//postprocess with cgroups and percentages
	last, ok := procStats.ProcsMap[status.Pid.ValueOr(0)]
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
	status.SampleTime = time.Now()
	if ok {
		cpuTotalPctNorm, cpuTotalPct, cpuTotalValue := GetProcCPUPercentage(last, status)
		status.CPU.Total.Norm.Pct = opt.FloatWith(cpuTotalPctNorm)
		status.CPU.Total.Pct = opt.FloatWith(cpuTotalPct)
		status.CPU.Total.Value = opt.FloatWith(cpuTotalValue)
	}

	return status, true, nil
}

func (procStats *Stats) cacheCmdLine(in ProcState) ProcState {
	if previousProc, ok := procStats.ProcsMap[in.Pid.ValueOr(0)]; ok {
		if procStats.CacheCmdLine {
			cmdline := previousProc.Cmdline
			in.Cmdline = cmdline
		}
		env := previousProc.Env
		in.Env = env
	}
	return in
}

// GetProcMemPercentage returns process memory usage as a percent of total memory usage
func GetProcMemPercentage(proc *ProcState, totalPhyMem uint64) opt.Float {
	if totalPhyMem == 0 {
		return opt.NewFloatNone()
	}

	perc := (float64(proc.Memory.Rss.Bytes.ValueOr(0)) / float64(totalPhyMem))

	return opt.FloatWith(common.Round(perc, 4))
}

func getProcState(b byte) string {
	switch b {
	case 'S':
		return "sleeping"
	case 'R':
		return "running"
	case 'D':
		return "idle"
	case 'T':
		return "stopped"
	case 'Z':
		return "zombie"
	}
	return "unknown"
}

// return a formatted MapStr of the process metrics
func (procStats *Stats) getProcessEvent(process *ProcState) (common.MapStr, error) {

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

	process.Memory.Rss.Pct = GetProcMemPercentage(process, totalPhyMem)

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

// GetProcCPUPercentage returns the percentage of total CPU time consumed by
// the process during the period between the given samples. Two percentages are
// returned (these must be multiplied by 100). The first is a normalized based
// on the number of cores such that the value ranges on [0, 1]. The second is
// not normalized and the value ranges on [0, number_of_cores].
//
// Implementation note: The total system CPU time (including idle) is not
// provided so this method will resort to using the difference in wall-clock
// time multiplied by the number of cores as the total amount of CPU time
// available between samples. This could result in incorrect percentages if the
// wall-clock is adjusted (prior to Go 1.9) or the machine is suspended.
func GetProcCPUPercentage(s0, s1 ProcState) (float64, float64, float64) {
	// Skip if we're missing the total ticks
	if s0.CPU.Total.Ticks.IsZero() || s1.CPU.Total.Ticks.IsZero() {
		return 0, 0, 0
	}

	timeDelta := s1.SampleTime.Sub(s0.SampleTime)
	timeDeltaMillis := timeDelta / time.Millisecond
	totalCPUDeltaMillis := int64(s1.CPU.Total.Ticks.ValueOr(0) - s0.CPU.Total.Ticks.ValueOr(0))

	pct := float64(totalCPUDeltaMillis) / float64(timeDeltaMillis)
	normalizedPct := pct / float64(numcpu.NumCPU())
	return common.Round(normalizedPct, common.DefaultDecimalPlacesCount),
		common.Round(pct, common.DefaultDecimalPlacesCount),
		common.Round(float64(s1.CPU.Total.Ticks.ValueOr(0)), common.DefaultDecimalPlacesCount)

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

// isProcessInSlice looks up proc in the processes slice and returns if
// found or not
func isProcessInSlice(processes []ProcState, proc *ProcState) bool {
	for _, p := range processes {
		if p.Pid == proc.Pid {
			return true
		}
	}
	return false
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

// unixTimeMsToTime converts a unix time given in milliseconds since Unix epoch
// to a common.Time value.
func unixTimeMsToTime(unixTimeMs uint64) string {
	return common.Time(time.Unix(0, int64(unixTimeMs*1000000))).String()
}
