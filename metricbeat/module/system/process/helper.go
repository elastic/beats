// +build darwin freebsd linux windows

package process

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/match"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/module/system"
	"github.com/elastic/beats/metricbeat/module/system/memory"
	sigar "github.com/elastic/gosigar"
)

type ProcsMap map[int]*Process

type Process struct {
	Pid      int    `json:"pid"`
	Ppid     int    `json:"ppid"`
	Pgid     int    `json:"pgid"`
	Name     string `json:"name"`
	Username string `json:"username"`
	State    string `json:"state"`
	CmdLine  string `json:"cmdline"`
	Mem      sigar.ProcMem
	Cpu      sigar.ProcTime
	Ctime    time.Time
	FD       sigar.ProcFDUsage
	Env      common.MapStr
}

type ProcStats struct {
	Procs        []string
	ProcsMap     ProcsMap
	CpuTicks     bool
	EnvWhitelist []string

	procRegexps []match.Matcher // List of regular expressions used to whitelist processes.
	envRegexps  []match.Matcher // List of regular expressions used to whitelist env vars.
}

// newProcess creates a new Process object and initializes it with process
// state information. If the process's command line and environment variables
// are known they should be passed in to avoid re-fetching the information.
func newProcess(pid int, cmdline string, env common.MapStr) (*Process, error) {
	state := sigar.ProcState{}
	if err := state.Get(pid); err != nil {
		return nil, fmt.Errorf("error getting process state for pid=%d: %v", pid, err)
	}

	proc := Process{
		Pid:      pid,
		Ppid:     state.Ppid,
		Pgid:     state.Pgid,
		Name:     state.Name,
		Username: state.Username,
		State:    getProcState(byte(state.State)),
		CmdLine:  cmdline,
		Ctime:    time.Now(),
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
		memStat, err := memory.GetMemory()
		if err != nil {
			logp.Warn("Getting memory details: %v", err)
			return 0
		}
		totalPhyMem = memStat.Mem.Total
	}

	perc := (float64(proc.Mem.Resident) / float64(totalPhyMem))

	return system.Round(perc, .5, 4)
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

func (procStats *ProcStats) GetProcessEvent(process *Process, last *Process) common.MapStr {
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

	if len(process.Env) > 0 {
		proc["env"] = process.Env
	}

	proc["cpu"] = common.MapStr{
		"total": common.MapStr{
			"pct": GetProcCpuPercentage(last, process),
		},
		"start_time": unixTimeMsToTime(process.Cpu.StartTime),
	}

	if procStats.CpuTicks {
		proc.Put("cpu.user", process.Cpu.User)
		proc.Put("cpu.system", process.Cpu.Sys)
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

func GetProcCpuPercentage(last *Process, current *Process) float64 {

	if last != nil && current != nil {

		dCPU := int64(current.Cpu.Total - last.Cpu.Total)
		dt := float64(current.Ctime.Sub(last.Ctime).Nanoseconds()) / float64(1e6) // in milliseconds
		perc := float64(dCPU) / dt

		return system.Round(perc, .5, 4)
	}
	return 0
}

func (procStats *ProcStats) MatchProcess(name string) bool {

	for _, reg := range procStats.procRegexps {
		if reg.MatchString(name) {
			return true
		}
	}
	return false
}

func (procStats *ProcStats) InitProcStats() error {

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

func (procStats *ProcStats) GetProcStats() ([]common.MapStr, error) {

	if len(procStats.Procs) == 0 {
		return nil, nil
	}

	pids, err := Pids()
	if err != nil {
		logp.Warn("Getting the list of pids: %v", err)
		return nil, err
	}

	processes := []common.MapStr{}
	newProcs := make(ProcsMap, len(pids))

	for _, pid := range pids {
		var cmdline string
		var env common.MapStr
		if previousProc := procStats.ProcsMap[pid]; previousProc != nil {
			cmdline = previousProc.CmdLine
			env = previousProc.Env
		}

		process, err := newProcess(pid, cmdline, env)
		if err != nil {
			logp.Debug("metricbeat", "Skip process pid=%d: %v", pid, err)
			continue
		}

		if procStats.MatchProcess(process.Name) {
			err = process.getDetails(procStats.isWhitelistedEnvVar)
			if err != nil {
				logp.Err("Error getting process details. pid=%d: %v", process.Pid, err)
				continue
			}

			newProcs[process.Pid] = process

			last := procStats.ProcsMap[process.Pid]
			proc := procStats.GetProcessEvent(process, last)

			processes = append(processes, proc)
		}
	}

	procStats.ProcsMap = newProcs
	return processes, nil
}

// isWhitelistedEnvVar returns true if the given variable name is a match for
// the whitelist. If the whitelist is empty it returns false.
func (p ProcStats) isWhitelistedEnvVar(varName string) bool {
	if len(p.envRegexps) == 0 {
		return false
	}

	for _, p := range p.envRegexps {
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
