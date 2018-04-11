// +build darwin freebsd linux windows

package process

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/match"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/metric/system/memory"
	sigar "github.com/elastic/gosigar"
)

// NumCPU is the number of CPUs of the host
var NumCPU = runtime.NumCPU()

// ProcsMap is a map where the keys are the names of processes and the value is the Process with that name
type ProcsMap map[int]*Process

// Process is the structure which holds the information of a process running on the host.
// It includes pid, gid and it interacts with gosigar to fetch process data from the host.
type Process struct {
	Pid             int    `json:"pid"`
	Ppid            int    `json:"ppid"`
	Pgid            int    `json:"pgid"`
	Name            string `json:"name"`
	Username        string `json:"username"`
	State           string `json:"state"`
	CmdLine         string `json:"cmdline"`
	Cwd             string `json:"cwd"`
	Mem             sigar.ProcMem
	Cpu             sigar.ProcTime
	SampleTime      time.Time
	FD              sigar.ProcFDUsage
	Env             common.MapStr
	cpuSinceStart   float64
	cpuTotalPct     float64
	cpuTotalPctNorm float64
}

// Stats stores the stats of processes on the host.
type Stats struct {
	Procs        []string
	ProcsMap     ProcsMap
	CpuTicks     bool
	EnvWhitelist []string
	CacheCmdLine bool
	IncludeTop   IncludeTopConfig

	procRegexps []match.Matcher // List of regular expressions used to whitelist processes.
	envRegexps  []match.Matcher // List of regular expressions used to whitelist env vars.
}

// Ticks of CPU for a process
type Ticks struct {
	User   uint64
	System uint64
	Total  uint64
}

// newProcess creates a new Process object and initializes it with process
// state information. If the process's command line and environment variables
// are known they should be passed in to avoid re-fetching the information.
func newProcess(pid int, cmdline string, env common.MapStr) (*Process, error) {
	state := sigar.ProcState{}
	if err := state.Get(pid); err != nil {
		return nil, fmt.Errorf("error getting process state for pid=%d: %v", pid, err)
	}

	exe := sigar.ProcExe{}
	if err := exe.Get(pid); err != nil && !sigar.IsNotImplemented(err) && !os.IsPermission(err) && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error getting process exe for pid=%d: %v", pid, err)
	}

	proc := Process{
		Pid:      pid,
		Ppid:     state.Ppid,
		Pgid:     state.Pgid,
		Name:     state.Name,
		Username: state.Username,
		State:    getProcState(byte(state.State)),
		CmdLine:  cmdline,
		Cwd:      exe.Cwd,
		Env:      env,
	}

	return &proc, nil
}

// getDetails fetches CPU, memory, FD usage, command line arguments, and
// environment variables for the process. The envPredicate parameter is an
// optional predicate function that should return true if an environment
// variable should be saved with the process. If the argument is nil then all
// environment variables are stored.
func (proc *Process) getDetails(envPredicate func(string) bool) error {
	proc.SampleTime = time.Now()

	proc.Mem = sigar.ProcMem{}
	if err := proc.Mem.Get(proc.Pid); err != nil {
		return fmt.Errorf("error getting process mem for pid=%d: %v", proc.Pid, err)
	}

	proc.Cpu = sigar.ProcTime{}
	if err := proc.Cpu.Get(proc.Pid); err != nil {
		return fmt.Errorf("error getting process cpu time for pid=%d: %v", proc.Pid, err)
	}

	if proc.CmdLine == "" {
		args := sigar.ProcArgs{}
		if err := args.Get(proc.Pid); err != nil && !sigar.IsNotImplemented(err) {
			return fmt.Errorf("error getting process arguments for pid=%d: %v", proc.Pid, err)
		}
		proc.CmdLine = strings.Join(args.List, " ")
	}

	if fd, err := getProcFDUsage(proc.Pid); err != nil {
		return fmt.Errorf("error getting process file descriptor usage for pid=%d: %v", proc.Pid, err)
	} else if fd != nil {
		proc.FD = *fd
	}

	if proc.Env == nil {
		proc.Env = common.MapStr{}
		if err := getProcEnv(proc.Pid, proc.Env, envPredicate); err != nil {
			return fmt.Errorf("error getting process environment variables for pid=%d: %v", proc.Pid, err)
		}
	}

	return nil
}

// getProcFDUsage returns file descriptor usage information for the process
// identified by the given PID. If the feature is not implemented then nil
// is returned with no error. If there is a permission error while reading the
// data then  nil is returned with no error (/proc/[pid]/fd requires root
// permissions). Any other errors that occur are returned.
func getProcFDUsage(pid int) (*sigar.ProcFDUsage, error) {
	// It's not possible to collect FD usage from other processes on FreeBSD
	// due to linprocfs not exposing the information.
	if runtime.GOOS == "freebsd" && pid != os.Getpid() {
		return nil, nil
	}

	fd := sigar.ProcFDUsage{}
	if err := fd.Get(pid); err != nil {
		switch {
		case sigar.IsNotImplemented(err):
			return nil, nil
		case os.IsPermission(err):
			return nil, nil
		default:
			return nil, err
		}
	}

	return &fd, nil
}

// getProcEnv gets the process's environment variables and writes them to the
// out parameter. It handles ErrNotImplemented and permission errors. Any other
// errors are returned.
//
// The filter function should return true if a given environment variable should
// be added to the out parameter.
//
// On Linux you must be root to read other processes' environment variables.
func getProcEnv(pid int, out common.MapStr, filter func(v string) bool) error {
	env := &sigar.ProcEnv{}
	if err := env.Get(pid); err != nil {
		switch {
		case sigar.IsNotImplemented(err):
			return nil
		case os.IsPermission(err):
			return nil
		default:
			return err
		}
	}

	for k, v := range env.Vars {
		if filter == nil || filter(k) {
			out[k] = v
		}
	}
	return nil
}

func GetProcMemPercentage(proc *Process, totalPhyMem uint64) float64 {
	// in unit tests, total_phymem is set to a value greater than zero
	if totalPhyMem == 0 {
		memStat, err := memory.Get()
		if err != nil {
			logp.Warn("Getting memory details: %v", err)
			return 0
		}
		totalPhyMem = memStat.Mem.Total
	}

	perc := (float64(proc.Mem.Resident) / float64(totalPhyMem))

	return common.Round(perc, 4)
}

func Pids() ([]int, error) {
	pids := sigar.ProcList{}
	err := pids.Get()
	if err != nil {
		return nil, err
	}
	return pids.List, nil
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

// GetOwnResourceUsageTimeInMillis return the user and system CPU usage time in milliseconds
func GetOwnResourceUsageTimeInMillis() (int64, int64, error) {
	r := sigar.Rusage{}
	err := r.Get(0)
	if err != nil {
		return 0, 0, err
	}

	uTime := int64(r.Utime / time.Millisecond)
	sTime := int64(r.Stime / time.Millisecond)

	return uTime, sTime, nil
}

func (procStats *Stats) getProcessEvent(process *Process) common.MapStr {
	proc := common.MapStr{
		"pid":      process.Pid,
		"ppid":     process.Ppid,
		"pgid":     process.Pgid,
		"name":     process.Name,
		"state":    process.State,
		"username": process.Username,
		"memory": common.MapStr{
			"size": process.Mem.Size,
			"rss": common.MapStr{
				"bytes": process.Mem.Resident,
				"pct":   GetProcMemPercentage(process, 0 /* read total mem usage */),
			},
			"share": process.Mem.Share,
		},
	}

	if process.CmdLine != "" {
		proc["cmdline"] = process.CmdLine
	}

	if process.Cwd != "" {
		proc["cwd"] = process.Cwd
	}

	if len(process.Env) > 0 {
		proc["env"] = process.Env
	}

	proc["cpu"] = common.MapStr{
		"total": common.MapStr{
			"value": process.cpuSinceStart,
			"pct":   process.cpuTotalPct,
			"norm": common.MapStr{
				"pct": process.cpuTotalPctNorm,
			},
		},
		"start_time": unixTimeMsToTime(process.Cpu.StartTime),
	}

	if procStats.CpuTicks {
		proc.Put("cpu.user.ticks", process.Cpu.User)
		proc.Put("cpu.system.ticks", process.Cpu.Sys)
		proc.Put("cpu.total.ticks", process.Cpu.Total)
	}

	if process.FD != (sigar.ProcFDUsage{}) {
		proc["fd"] = common.MapStr{
			"open": process.FD.Open,
			"limit": common.MapStr{
				"soft": process.FD.SoftLimit,
				"hard": process.FD.HardLimit,
			},
		}
	}

	return proc
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
func GetProcCPUPercentage(s0, s1 *Process) (normalizedPct, pct, totalPct float64) {
	if s0 != nil && s1 != nil {
		timeDelta := s1.SampleTime.Sub(s0.SampleTime)
		timeDeltaMillis := timeDelta / time.Millisecond
		totalCPUDeltaMillis := int64(s1.Cpu.Total - s0.Cpu.Total)

		pct := float64(totalCPUDeltaMillis) / float64(timeDeltaMillis)
		normalizedPct := pct / float64(NumCPU)

		return common.Round(normalizedPct, common.DefaultDecimalPlacesCount),
			common.Round(pct, common.DefaultDecimalPlacesCount),
			common.Round(float64(s1.Cpu.Total), common.DefaultDecimalPlacesCount)
	}
	return 0, 0, 0
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

// Init initizalizes a Stats instance. It returns erros if the provided process regexes
// cannot be compiled.
func (procStats *Stats) Init() error {
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

	return nil
}

// Get fetches process data which matches the provided regexes from the host.
func (procStats *Stats) Get() ([]common.MapStr, error) {
	if len(procStats.Procs) == 0 {
		return nil, nil
	}

	pids, err := Pids()
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch the list of PIDs")
	}

	var processes []Process
	newProcs := make(ProcsMap, len(pids))

	for _, pid := range pids {
		process := procStats.getSingleProcess(pid, newProcs)
		if process == nil {
			continue
		}
		processes = append(processes, *process)
	}
	procStats.ProcsMap = newProcs

	processes = procStats.includeTopProcesses(processes)
	logp.Debug("processes", "Filtered top processes down to %d processes", len(processes))

	procs := make([]common.MapStr, 0, len(processes))
	for _, process := range processes {
		proc := procStats.getProcessEvent(&process)
		procs = append(procs, proc)
	}

	return procs, nil
}

// GetOne fetches process data for a given PID if its name matches the regexes provided from the host.
func (procStats *Stats) GetOne(pid int) (common.MapStr, error) {
	if len(procStats.Procs) == 0 {
		return nil, nil
	}

	newProcs := make(ProcsMap, 1)
	p := procStats.getSingleProcess(pid, newProcs)
	if p == nil {
		return nil, fmt.Errorf("cannot find matching process for pid=%d", pid)
	}

	e := procStats.getProcessEvent(p)
	procStats.ProcsMap = newProcs

	return e, nil
}

func (procStats *Stats) getSingleProcess(pid int, newProcs ProcsMap) *Process {
	var cmdline string
	var env common.MapStr
	if previousProc := procStats.ProcsMap[pid]; previousProc != nil {
		if procStats.CacheCmdLine {
			cmdline = previousProc.CmdLine
		}
		env = previousProc.Env
	}

	process, err := newProcess(pid, cmdline, env)
	if err != nil {
		logp.Debug("processes", "Skip process pid=%d: %v", pid, err)
		return nil
	}

	if !procStats.matchProcess(process.Name) {
		logp.Debug("processes", "Process name does not matches the provided regex; pid=%d; name=%s: %v", pid, process.Name, err)
		return nil
	}

	err = process.getDetails(procStats.isWhitelistedEnvVar)
	if err != nil {
		logp.Err("Error getting process details. pid=%d: %v", process.Pid, err)
		return nil
	}

	newProcs[process.Pid] = process
	last := procStats.ProcsMap[process.Pid]
	process.cpuTotalPctNorm, process.cpuTotalPct, process.cpuSinceStart = GetProcCPUPercentage(last, process)
	return process
}

func (procStats *Stats) includeTopProcesses(processes []Process) []Process {
	if !procStats.IncludeTop.Enabled ||
		(procStats.IncludeTop.ByCPU == 0 && procStats.IncludeTop.ByMemory == 0) {

		return processes
	}

	var result []Process
	if procStats.IncludeTop.ByCPU > 0 {
		numProcs := procStats.IncludeTop.ByCPU
		if len(processes) < procStats.IncludeTop.ByCPU {
			numProcs = len(processes)
		}

		sort.Slice(processes, func(i, j int) bool {
			return processes[i].cpuTotalPct > processes[j].cpuTotalPct
		})
		result = append(result, processes[:numProcs]...)
	}

	if procStats.IncludeTop.ByMemory > 0 {
		numProcs := procStats.IncludeTop.ByMemory
		if len(processes) < procStats.IncludeTop.ByMemory {
			numProcs = len(processes)
		}

		sort.Slice(processes, func(i, j int) bool {
			return processes[i].Mem.Resident > processes[j].Mem.Resident
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
func isProcessInSlice(processes []Process, proc *Process) bool {
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
func unixTimeMsToTime(unixTimeMs uint64) common.Time {
	return common.Time(time.Unix(0, int64(unixTimeMs*1000000)))
}
