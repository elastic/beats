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

//go:build (darwin && cgo) || freebsd || linux || windows || aix

package process

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"syscall"
	"time"

	psutil "github.com/shirou/gopsutil/v4/process"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-libs/transform/typeconv"
	"github.com/elastic/elastic-agent-system-metrics/metric"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/network"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
	"github.com/elastic/go-sysinfo"
	sysinfotypes "github.com/elastic/go-sysinfo/types"
)

var errFetchingPIDs = "error fetching PID metrics for %d processes, most likely a \"permission denied\" error. Enable debug logging to determine the exact cause."

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
		return nil, fmt.Errorf("error initializing process collectors: %w", err)
	}

	// actually fetch the PIDs from the OS-specific code
	pidMap, plist, err := init.FetchPids()
	if err != nil && !isNonFatal(err) {
		return nil, fmt.Errorf("error gathering PIDs: %w", err)
	}
	failedPIDs := extractFailedPIDs(pidMap)
	if err != nil && len(failedPIDs) > 0 {
		init.logger.Debugf("error fetching process metrics: %v", err)
		return plist, NonFatalErr{Err: fmt.Errorf(errFetchingPIDs, len(failedPIDs))}
	}
	return plist, toNonFatal(err)
}

// GetPIDState returns the state of a given PID
// It will return ErrProcNotExist if the process was not found.
func GetPIDState(hostfs resolve.Resolver, pid int) (PidState, error) {
	// This library still doesn't have a good cross-platform way to distinguish between "does not eixst" and other process errors.
	// This is a fairly difficult problem to solve in a cross-platform way
	exists, err := psutil.PidExistsWithContext(context.Background(), int32(pid))
	if err != nil {
		return "", fmt.Errorf("Error trying to find process: %d: %w", pid, err)
	}
	if !exists {
		return "", ErrProcNotExist
	}
	// GetInfoForPid will return the smallest possible dataset for a PID
	procState, err := GetInfoForPid(hostfs, pid)
	if err != nil {
		return "", fmt.Errorf("error getting state info for pid %d: %w", pid, err)
	}

	return procState.State, nil
}

// Get fetches the configured processes and returns a list of formatted events and root ECS fields
func (procStats *Stats) Get() ([]mapstr.M, []mapstr.M, error) {
	// If the user hasn't configured any kind of process glob, return
	if len(procStats.Procs) == 0 {
		return nil, nil, nil
	}

	// actually fetch the PIDs from the OS-specific code
	pidMap, plist, wrappedErr := procStats.FetchPids()

	if wrappedErr != nil && !isNonFatal(wrappedErr) {
		return nil, nil, fmt.Errorf("error gathering PIDs: %w", wrappedErr)
	}
	failedPIDs := extractFailedPIDs(pidMap)
	// We use this to track processes over time.
	procStats.ProcsMap.SetMap(pidMap)

	// filter the process list that will be passed down to users
	plist = procStats.includeTopProcesses(plist)

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

	// Format the list to the MapStr type used by the outputs
	procs := make([]mapstr.M, 0, len(plist))
	rootEvents := make([]mapstr.M, 0, len(plist))

	for _, process := range plist {
		process := process
		// Add the RSS pct memory first
		process.Memory.Rss.Pct = GetProcMemPercentage(process, totalPhyMem)
		// Create the root event
		rootMap := processRootEvent(&process)

		proc, err := procStats.getProcessEvent(&process)
		if err != nil {
			return nil, nil, fmt.Errorf("error converting process for pid %d: %w", process.Pid.ValueOr(0), err)
		}

		procs = append(procs, proc)
		rootEvents = append(rootEvents, rootMap)
	}
	if len(failedPIDs) > 0 {
		procStats.logger.Debugf("error fetching process metrics: %v", wrappedErr)
		return procs, rootEvents, NonFatalErr{Err: fmt.Errorf(errFetchingPIDs, len(failedPIDs))}
	}
	return procs, rootEvents, nil
}

// GetOne fetches process data for a given PID if its name matches the regexes provided from the host.
func (procStats *Stats) GetOne(pid int) (mapstr.M, error) {
	pidStat, _, err := procStats.pidFill(pid, false)
	if err != nil && !isNonFatal(err) {
		return nil, fmt.Errorf("error fetching PID %d: %w", pid, err)
	}

	procStats.ProcsMap.SetPid(pid, pidStat)

	return procStats.getProcessEvent(&pidStat)
}

// GetOneRootEvent is the same as `GetOne()` but it returns an
// event formatted as expected by ECS
func (procStats *Stats) GetOneRootEvent(pid int) (mapstr.M, mapstr.M, error) {
	pidStat, _, wrappedErr := procStats.pidFill(pid, false)
	if wrappedErr != nil && !isNonFatal(wrappedErr) {
		return nil, nil, fmt.Errorf("error fetching PID %d: %w", pid, wrappedErr)
	}

	procStats.ProcsMap.SetPid(pid, pidStat)

	procMap, err := procStats.getProcessEvent(&pidStat)
	if err != nil {
		return nil, nil, fmt.Errorf("error formatting process %d: %w", pid, err)
	}

	rootMap := processRootEvent(&pidStat)

	return procMap, rootMap, toNonFatal(wrappedErr)
}

// GetSelf gets process info for the beat itself
// Be advised that if you call this method on a Stats object that was created with an alternate
// `Hostfs` setting, this method will return data for that pid as it exists on that hostfs.
// For example, if called from inside a container with a `hostfs` path for the container host,
// the PID in the ProcState object will be the PID as the host sees it.
func (procStats *Stats) GetSelf() (ProcState, error) {
	self, err := GetSelfPid(procStats.Hostfs)
	if err != nil {
		return ProcState{}, fmt.Errorf("error finding PID: %w", err)
	}

	pidStat, _, err := procStats.pidFill(self, false)
	if err != nil && !isNonFatal(err) {
		return ProcState{}, fmt.Errorf("error fetching PID %d: %w", self, err)
	}

	procStats.ProcsMap.SetPid(self, pidStat)

	return pidStat, toNonFatal(err)
}

// pidIter wraps a few lines of generic code that all OS-specific FetchPids() functions must call.
// this also handles the process of adding to the maps/lists in order to limit the code duplication in all the OS implementations
// NOTE: this method will sometimes return a NonFatalError{} wrapper for errors that can optionally be ignored.
func (procStats *Stats) pidIter(pid int, procMap ProcsMap, proclist []ProcState) (ProcsMap, []ProcState, error) {
	status, saved, err := procStats.pidFill(pid, true)
	var nonFatalErr error
	if err != nil {
		procMap[pid] = ProcState{Failed: true}
		if !errors.Is(err, NonFatalErr{}) {
			procStats.logger.Debugf("Error fetching PID info for %d, skipping: %s", pid, err)
			// While monitoring a set of processes, some processes might get killed after we get all the PIDs
			// So, there's no need to capture "process not found" error.
			if errors.Is(err, syscall.ESRCH) {
				return procMap, proclist, nil
			}
			return procMap, proclist, err
		}
		nonFatalErr = fmt.Errorf("error for pid %d: %w", pid, err)
		procStats.logger.Debugf(err.Error())
	}
	if !saved {
		procStats.logger.Debugf("Process name does not match the provided regex; PID=%d; name=%s", pid, status.Name)
		return procMap, proclist, nonFatalErr
	}
	procMap[pid] = status
	proclist = append(proclist, status)

	return procMap, proclist, nonFatalErr
}

// pidFill is an entrypoint used by OS-specific code to fill out a pid.
// This in turn calls various OS-specific code to fill out the various bits of PID data
// This is done to minimize the code duplication between different OS implementations
// The second return value will only be false if an event has been filtered out.
func (procStats *Stats) pidFill(pid int, filter bool) (ProcState, bool, error) {
	// Fetch proc state so we can get the name for filtering based on user's filter.
	var wrappedErr error
	// OS-specific entrypoint, get basic info so we can at least run matchProcess
	status, err := GetInfoForPid(procStats.Hostfs, pid)
	if err != nil {
		return status, true, fmt.Errorf("GetInfoForPid failed for pid %d: %w", pid, err)
	}
	if procStats.skipExtended {
		return status, true, nil
	}

	// Some OSes use the cache to avoid expensive system calls,
	// cacheCmdLine reads from the cache.
	status = procStats.cacheCmdLine(status)

	// Filter based on user-supplied func
	if filter {
		if !procStats.matchProcess(status.Name) {
			return status, false, nil
		}
	}

	// If we've passed the filter, continue to fill out the rest of the metrics
	status, err = FillPidMetrics(procStats.Hostfs, pid, status, procStats.isWhitelistedEnvVar)
	if err != nil {
		if !errors.Is(err, NonFatalErr{}) {
			return status, true, fmt.Errorf("FillPidMetrics failed for PID %d: %w", pid, err)
		}
		wrappedErr = errors.Join(wrappedErr, err)
		procStats.logger.Debugf(wrappedErr.Error())
	}

	if status.CPU.Total.Ticks.Exists() {
		status.CPU.Total.Value = opt.FloatWith(metric.Round(float64(status.CPU.Total.Ticks.ValueOr(0))))
	}

	// postprocess with cgroups and percentages
	last, ok := procStats.ProcsMap.GetPid(status.Pid.ValueOr(0))
	status.SampleTime = time.Now()
	if ok {
		status = GetProcCPUPercentage(last, status)
	}

	if procStats.EnableCgroups {
		cgStats, err := procStats.cgroups.GetStatsForPid(status.Pid.ValueOr(0))
		if err != nil {
			procStats.logger.Debugf("Non-fatal error fetching cgroups metrics for pid %d, metrics are valid but partial: %s", pid, err)
		} else {
			status.Cgroup = cgStats
			if ok {
				status.Cgroup.FillPercentages(last.Cgroup, status.SampleTime, last.SampleTime)
			}
		}

	} // end cgroups processor

	status, err = FillMetricsRequiringMoreAccess(pid, status)
	if err != nil {
		procStats.logger.Debugf("error calling FillMetricsRequiringMoreAccess for pid %d: %w", pid, err)
	}

	// Generate `status.Cmdline` here for compatibility because on Windows
	// `status.Args` is set by `FillMetricsRequiringMoreAccess`.
	if len(status.Args) > 0 && status.Cmdline == "" {
		status.Cmdline = strings.Join(status.Args, " ")
	}

	// network data
	if procStats.EnableNetwork {
		procHandle, err := sysinfo.Process(pid)
		// treat this as a soft error
		if err != nil {
			procStats.logger.Debugf("error initializing process handler for pid %d while trying to fetch network data: %w", pid, err)
		} else {
			procNet, ok := procHandle.(sysinfotypes.NetworkCounters)
			if ok {
				status.Network, err = procNet.NetworkCounters()
				if err != nil {
					procStats.logger.Debugf("error fetching network counters for process %d: %w", pid, err)
				}
			}
		}
	}

	return status, true, wrappedErr
}

// cacheCmdLine fills out Env and arg metrics from any stored previous metrics for the pid
func (procStats *Stats) cacheCmdLine(in ProcState) ProcState {
	if previousProc, ok := procStats.ProcsMap.GetPid(in.Pid.ValueOr(0)); ok {
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
func (procStats *Stats) getProcessEvent(process *ProcState) (mapstr.M, error) {

	// Remove CPUTicks if needed
	if !procStats.CPUTicks {
		process.CPU.User.Ticks = opt.NewUintNone()
		process.CPU.System.Ticks = opt.NewUintNone()
		process.CPU.Total.Ticks = opt.NewUintNone()
	}

	proc := mapstr.M{}
	err := typeconv.Convert(&proc, process)

	if procStats.EnableNetwork && process.Network != nil {
		proc["network"] = network.MapProcNetCountersWithFilter(process.Network, procStats.NetworkMetrics)
	}

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
			proc := proc
			if !isProcessInSlice(result, &proc) {
				result = append(result, proc)
			}
		}
	}

	return result
}

// isWhitelistedEnvVar returns true if the given variable name is a match for
// the whitelist. If the whitelist is empty it returns false.
func (procStats *Stats) isWhitelistedEnvVar(varName string) bool {
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

func extractFailedPIDs(procMap ProcsMap) []int {
	list := make([]int, 0)
	for pid, state := range procMap {
		if state.Failed {
			list = append(list, pid)
			// delete the failed state so we don't return the state to caller
			delete(procMap, pid)
		}
	}
	return list
}
