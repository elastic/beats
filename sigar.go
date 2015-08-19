package main

import (
	"fmt"
	"time"

	"github.com/elastic/gosigar"
	"github.com/elastic/libbeat/logp"
)

type SystemLoad struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

type CpuTimes struct {
	User    uint64 `json:"user"`
	Nice    uint64 `json:"nice"`
	System  uint64 `json:"system"`
	Idle    uint64 `json:"idle"`
	IOWait  uint64 `json:"iowait"`
	Irq     uint64 `json:"irq"`
	SoftIrq uint64 `json:"softirq"`
	Steal   uint64 `json:"steal"`
}

type MemStat struct {
	Total      uint64 `json:"total"`
	Used       uint64 `json:"used"`
	Free       uint64 `json:"free"`
	ActualUsed uint64 `json:"actual_used"`
	ActualFree uint64 `json:"actual_free"`
}

type ProcMemStat struct {
	Size     uint64 `json:"size"`
	Resident uint64 `json:"rss"`
	Share    uint64 `json:"share"`
}

type ProcCpuTime struct {
	User    uint64  `json:"user"`
	Percent float64 `json:"percent"`
	System  uint64  `json:"system"`
	Total   uint64  `json:"total"`
	Start   string  `json:"start_time"`
}

type Process struct {
	Pid         int         `json:"pid"`
	Ppid        int         `json:"ppid"`
	Name        string      `json:"name"`
	State       string      `json:"state"`
	Mem         ProcMemStat `json:"mem"`
	Cpu         ProcCpuTime `json:"cpu"`
	lastCPUTime time.Time
}

func (p *Process) String() string {

	return fmt.Sprintf("pid: %d, ppid: %d, name: %s, state: %s, mem: %s, cpu: %s",
		p.Pid, p.Ppid, p.Name, p.State, p.Mem.String(), p.Cpu.String())
}

func (m *ProcMemStat) String() string {

	return fmt.Sprintf("%d size, %d rss, %d share", m.Size, m.Resident, m.Share)
}

func (t *ProcCpuTime) String() string {
	return fmt.Sprintf("started at %s, %d total %.2f%%CPU, %d us, %d sys", t.Start, t.Total, t.Percent, t.User, t.System)

}

func (m *MemStat) String() string {

	return fmt.Sprintf("%d total, %d used, %d actual used, %d free, %d actual free", m.Total, m.Used, m.ActualUsed,
		m.Free, m.ActualFree)
}

func (t *SystemLoad) String() string {

	return fmt.Sprintf("%.2f %.2f %.2f", t.Load1, t.Load5, t.Load15)
}

func (t *CpuTimes) String() string {

	return fmt.Sprintf("%d user, %d system, %d nice, %d iddle, %d iowait, %d irq, %d softirq, %d steal",
		t.User, t.System, t.Nice, t.Idle, t.IOWait, t.Irq, t.SoftIrq, t.Steal)
}

func GetSystemLoad() (*SystemLoad, error) {

	concreteSigar := sigar.ConcreteSigar{}
	avg, err := concreteSigar.GetLoadAverage()
	if err != nil {
		return nil, err
	}
	logp.Debug("topbeat", "load %v\n", avg)

	return &SystemLoad{
		Load1:  avg.One,
		Load5:  avg.Five,
		Load15: avg.Fifteen,
	}, nil
}

func GetCpuTimes() (*CpuTimes, error) {

	cpu := sigar.Cpu{}
	err := cpu.Get()
	if err != nil {
		return nil, err
	}

	logp.Debug("topbeat", "cpu times %v\n", cpu)

	return &CpuTimes{
		User:    cpu.User,
		Nice:    cpu.Nice,
		System:  cpu.Sys,
		Idle:    cpu.Idle,
		IOWait:  cpu.Wait,
		Irq:     cpu.Irq,
		SoftIrq: cpu.SoftIrq,
		Steal:   cpu.Stolen,
	}, nil
}

func GetMemory() (*MemStat, error) {

	mem := sigar.Mem{}
	err := mem.Get()
	if err != nil {
		return nil, err
	}
	return &MemStat{
		Total:      mem.Total / 1024,
		Used:       mem.Used / 1024,
		Free:       mem.Free / 1024,
		ActualFree: mem.ActualFree / 1024,
		ActualUsed: mem.ActualUsed / 1024,
	}, nil
}

func GetSwap() (*MemStat, error) {

	swap := sigar.Swap{}
	err := swap.Get()
	if err != nil {
		return nil, err
	}
	return &MemStat{
		Total: swap.Total,
		Used:  swap.Used,
		Free:  swap.Free,
	}, nil

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

func GetProcess(pid int) (*Process, error) {

	state := sigar.ProcState{}
	mem := sigar.ProcMem{}
	cpu := sigar.ProcTime{}

	err := state.Get(pid)
	if err != nil {
		return nil, err
	}

	err = mem.Get(pid)
	if err != nil {
		return nil, err
	}

	err = cpu.Get(pid)
	if err != nil {
		return nil, err
	}

	proc := Process{
		Pid:   pid,
		Ppid:  state.Ppid,
		Name:  state.Name,
		State: getProcState(byte(state.State)),
		Mem: ProcMemStat{
			Size:     mem.Size / 1024,
			Resident: mem.Resident / 1024,
			Share:    mem.Share / 1024,
		},
		Cpu: ProcCpuTime{
			Start:  cpu.FormatStartTime(),
			Total:  cpu.Total,
			User:   cpu.User,
			System: cpu.Sys,
		},
	}

	proc.lastCPUTime = time.Now()
	return &proc, nil
}
