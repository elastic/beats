package beat

import (
	"fmt"
	"strings"
	"time"

	sigar "github.com/elastic/gosigar"
)

type SystemLoad struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

type CpuTimes struct {
	User          uint64  `json:"user"`
	UserPercent   float64 `json:"user_p"`
	Nice          uint64  `json:"nice"`
	System        uint64  `json:"system"`
	SystemPercent float64 `json:"system_p"`
	Idle          uint64  `json:"idle"`
	IOWait        uint64  `json:"iowait"`
	Irq           uint64  `json:"irq"`
	SoftIrq       uint64  `json:"softirq"`
	Steal         uint64  `json:"steal"`
}

type MemStat struct {
	Total             uint64  `json:"total"`
	Used              uint64  `json:"used"`
	Free              uint64  `json:"free"`
	UsedPercent       float64 `json:"used_p"`
	ActualUsed        uint64  `json:"actual_used"`
	ActualFree        uint64  `json:"actual_free"`
	ActualUsedPercent float64 `json:"actual_used_p"`
}

type SwapStat struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_p"`
}

type ProcMemStat struct {
	Size       uint64  `json:"size"`
	Rss        uint64  `json:"rss"`
	RssPercent float64 `json:"rss_p"`
	Share      uint64  `json:"share"`
}

type ProcCpuTime struct {
	User        uint64  `json:"user"`
	UserPercent float64 `json:"user_p"`
	System      uint64  `json:"system"`
	Total       uint64  `json:"total"`
	Start       string  `json:"start_time"`
}

type Process struct {
	Pid      int          `json:"pid"`
	Ppid     int          `json:"ppid"`
	Name     string       `json:"name"`
	Username string       `json:"username"`
	State    string       `json:"state"`
	CmdLine  string       `json:"cmdline"`
	Mem      *ProcMemStat `json:"mem"`
	Cpu      *ProcCpuTime `json:"cpu"`
	ctime    time.Time
}

type FileSystemStat struct {
	DevName     string  `json:"device_name"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_p"`
	Free        uint64  `json:"free"`
	Avail       uint64  `json:"avail"`
	Files       uint64  `json:"files"`
	FreeFiles   uint64  `json:"free_files"`
	Mount       string  `json:"mount_point"`
	ctime       time.Time
}

func (f *FileSystemStat) String() string {

	return fmt.Sprintf("device name: %s, total: %d, used %d, used pct %.2f, free: %d, avail: %d, files: %d, free files: %d, mount: %s",
		f.DevName, f.Total, f.Used, f.UsedPercent, f.Free, f.Avail, f.Files, f.FreeFiles, f.Mount)
}

func (p *Process) String() string {

	return fmt.Sprintf("pid: %d, ppid: %d, name: %s, state: %s, mem: %s, cpu: %s",
		p.Pid, p.Ppid, p.Name, p.State, p.Mem.String(), p.Cpu.String())
}

func (m *ProcMemStat) String() string {

	return fmt.Sprintf("%d size, %d rss, %d share", m.Size, m.Rss, m.Share)
}

func (t *ProcCpuTime) String() string {
	return fmt.Sprintf("started at %s, %d total %.2f%%CPU, %d us, %d sys", t.Start, t.Total, t.UserPercent, t.User, t.System)

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

func GetCpuTimesList() ([]CpuTimes, error) {

	cpuList := sigar.CpuList{}
	err := cpuList.Get()
	if err != nil {
		return nil, err
	}

	cpuTimes := make([]CpuTimes, len(cpuList.List))

	for i, cpu := range cpuList.List {
		cpuTimes[i] = CpuTimes{
			User:    cpu.User,
			Nice:    cpu.Nice,
			System:  cpu.Sys,
			Idle:    cpu.Idle,
			IOWait:  cpu.Wait,
			Irq:     cpu.Irq,
			SoftIrq: cpu.SoftIrq,
			Steal:   cpu.Stolen,
		}
	}

	return cpuTimes, nil
}

func GetMemory() (*MemStat, error) {

	mem := sigar.Mem{}
	err := mem.Get()
	if err != nil {
		return nil, err
	}

	return &MemStat{
		Total:      mem.Total,
		Used:       mem.Used,
		Free:       mem.Free,
		ActualFree: mem.ActualFree,
		ActualUsed: mem.ActualUsed,
	}, nil
}

func GetSwap() (*SwapStat, error) {

	swap := sigar.Swap{}
	err := swap.Get()
	if err != nil {
		return nil, err
	}

	return &SwapStat{
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

// newProcess creates a new Process object based on the state information.
func newProcess(pid int) (*Process, error) {

	state := sigar.ProcState{}
	if err := state.Get(pid); err != nil {
		return nil, fmt.Errorf("error getting process state for pid=%d: %v", pid, err)
	}

	proc := Process{
		Pid:      pid,
		Ppid:     state.Ppid,
		Name:     state.Name,
		State:    getProcState(byte(state.State)),
		Username: state.Username,
		ctime:    time.Now(),
	}

	return &proc, nil
}

// getDetails fills in CPU, memory, and command line details for the process
func (proc *Process) getDetails(cmdline string) error {

	mem := sigar.ProcMem{}
	if err := mem.Get(proc.Pid); err != nil {
		return fmt.Errorf("error getting process mem for pid=%d: %v", proc.Pid, err)
	}
	proc.Mem = &ProcMemStat{
		Size:  mem.Size,
		Rss:   mem.Resident,
		Share: mem.Share,
	}

	cpu := sigar.ProcTime{}
	if err := cpu.Get(proc.Pid); err != nil {
		return fmt.Errorf("error getting process cpu time for pid=%d: %v", proc.Pid, err)
	}
	proc.Cpu = &ProcCpuTime{
		Start:  cpu.FormatStartTime(),
		Total:  cpu.Total,
		User:   cpu.User,
		System: cpu.Sys,
	}

	if cmdline == "" {
		args := sigar.ProcArgs{}
		if err := args.Get(proc.Pid); err != nil {
			return fmt.Errorf("error getting process arguments for pid=%d: %v", proc.Pid, err)
		}
		proc.CmdLine = strings.Join(args.List, " ")
	} else {
		proc.CmdLine = cmdline
	}

	return nil
}

func GetFileSystemList() ([]sigar.FileSystem, error) {

	fss := sigar.FileSystemList{}
	err := fss.Get()
	if err != nil {
		return nil, err
	}

	return fss.List, nil
}

func GetFileSystemStat(fs sigar.FileSystem) (*FileSystemStat, error) {

	stat := sigar.FileSystemUsage{}
	err := stat.Get(fs.DirName)
	if err != nil {
		return nil, err
	}

	filesystem := FileSystemStat{
		DevName:   fs.DevName,
		Total:     stat.Total,
		Free:      stat.Free,
		Avail:     stat.Avail,
		Used:      stat.Used,
		Files:     stat.Files,
		FreeFiles: stat.FreeFiles,
		Mount:     fs.DirName,
	}

	return &filesystem, nil
}
