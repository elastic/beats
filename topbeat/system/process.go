package system

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	sigar "github.com/elastic/gosigar"
)

type ProcsMap map[int]*Process

type Process struct {
	Pid      int    `json:"pid"`
	Ppid     int    `json:"ppid"`
	Name     string `json:"name"`
	Username string `json:"username"`
	State    string `json:"state"`
	CmdLine  string `json:"cmdline"`
	Mem      sigar.ProcMem
	Cpu      sigar.ProcTime
	Ctime    time.Time
}

type ProcStats struct {
	ProcStats bool
	Procs     []string
	ProcsMap  ProcsMap
}

func GetProcess(pid int, cmdline string) (*Process, error) {
	state := sigar.ProcState{}
	if err := state.Get(pid); err != nil {
		return nil, fmt.Errorf("error getting process state for pid=%d: %v", pid, err)
	}

	mem := sigar.ProcMem{}
	if err := mem.Get(pid); err != nil {
		return nil, fmt.Errorf("error getting process mem for pid=%d: %v", pid, err)
	}

	cpu := sigar.ProcTime{}
	if err := cpu.Get(pid); err != nil {
		return nil, fmt.Errorf("error getting process cpu time for pid=%d: %v", pid, err)
	}

	if cmdline == "" {
		args := sigar.ProcArgs{}
		if err := args.Get(pid); err != nil {
			return nil, fmt.Errorf("error getting process arguments for pid=%d: %v", pid, err)
		}
		cmdline = strings.Join(args.List, " ")
	}

	proc := Process{
		Pid:      pid,
		Ppid:     state.Ppid,
		Name:     state.Name,
		State:    getProcState(byte(state.State)),
		Username: state.Username,
		CmdLine:  cmdline,
		Mem:      mem,
		Cpu:      cpu,
		Ctime:    time.Now(),
	}

	return &proc, nil
}

func GetProcMemPercentage(proc *Process, total_phymem uint64) float64 {

	// in unit tests, total_phymem is set to a value greater than zero

	if total_phymem == 0 {
		memStat, err := GetMemory()
		if err != nil {
			logp.Warn("Getting memory details: %v", err)
			return 0
		}
		total_phymem = memStat.Mem.Total
	}

	perc := (float64(proc.Mem.Resident) / float64(total_phymem))

	return Round(perc, .5, 4)
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

func GetProcessEvent(process *Process, last *Process) common.MapStr {
	proc := common.MapStr{
		"pid":      process.Pid,
		"ppid":     process.Ppid,
		"name":     process.Name,
		"state":    process.State,
		"username": process.Username,
		"mem": common.MapStr{
			"size":  process.Mem.Size,
			"rss":   process.Mem.Resident,
			"rss_p": GetProcMemPercentage(process, 0 /* read total mem usage */),
			"share": process.Mem.Share,
		},
		"cpu": common.MapStr{
			"user":       process.Cpu.User,
			"system":     process.Cpu.Sys,
			"total":      process.Cpu.Total,
			"total_p":    GetProcCpuPercentage(last, process),
			"start_time": process.Cpu.FormatStartTime(),
		},
	}

	if process.CmdLine != "" {
		proc["cmdline"] = process.CmdLine
	}

	return proc
}

func GetProcCpuPercentage(last *Process, current *Process) float64 {

	if last != nil && current != nil {

		delta_proc := int64(current.Cpu.Total - last.Cpu.Total)
		delta_time := float64(current.Ctime.Sub(last.Ctime).Nanoseconds()) / float64(1e6) // in milliseconds
		perc := float64(delta_proc) / delta_time

		return Round(perc, .5, 4)
	}
	return 0
}

func (procStats *ProcStats) MatchProcess(name string) bool {

	for _, reg := range procStats.Procs {
		matched, _ := regexp.MatchString(reg, name)
		if matched {
			return true
		}
	}
	return false
}

func (procStats *ProcStats) InitProcStats() {

	procStats.ProcsMap = make(ProcsMap)

	if len(procStats.Procs) == 0 {
		return
	}

	pids, err := Pids()
	if err != nil {
		logp.Warn("Getting the initial list of pids: %v", err)
	}

	for _, pid := range pids {
		process, err := GetProcess(pid, "")
		if err != nil {
			logp.Debug("topbeat", "Skip process pid=%d: %v", pid, err)
			continue
		}
		procStats.ProcsMap[process.Pid] = process
	}
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
		if previousProc := procStats.ProcsMap[pid]; previousProc != nil {
			cmdline = previousProc.CmdLine
		}

		process, err := GetProcess(pid, cmdline)
		if err != nil {
			logp.Debug("topbeat", "Skip process pid=%d: %v", pid, err)
			continue
		}

		if procStats.MatchProcess(process.Name) {

			newProcs[process.Pid] = process

			last, ok := procStats.ProcsMap[process.Pid]
			if ok {
				procStats.ProcsMap[process.Pid] = process
			}
			proc := GetProcessEvent(process, last)

			processes = append(processes, proc)
		}
	}

	procStats.ProcsMap = newProcs
	return processes, nil
}

func (procStats *ProcStats) GetProcStatsEvents() ([]common.MapStr, error) {

	events := []common.MapStr{}

	processes, err := procStats.GetProcStats()
	if err != nil {
		return nil, err
	}

	for _, proc := range processes {
		event := common.MapStr{
			"@timestamp": common.Time(time.Now()),
			"type":       "process",
			"proc":       proc,
		}

		events = append(events, event)
	}

	return events, nil
}
