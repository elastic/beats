package system

import (
	"fmt"
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

	return Round(perc, .5, 2)
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
			"total_p":    GetProcCpuPercentage(process, last),
			"start_time": process.Cpu.FormatStartTime(),
		},
	}

	if process.CmdLine != "" {
		proc["cmdline"] = process.CmdLine
	}

	return proc
}

func GetProcCpuPercentage(last *Process, current *Process) float64 {

	if last != nil {

		delta_proc := (current.Cpu.User - last.Cpu.User) + (current.Cpu.Sys - last.Cpu.Sys)
		delta_time := current.Ctime.Sub(last.Ctime).Nanoseconds() / 1e6 // in milliseconds
		perc := float64(delta_proc) / float64(delta_time)

		return Round(perc, .5, 4)
	}
	return 0
}
