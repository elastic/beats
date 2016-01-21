package beat

import (
	"fmt"
	"strings"
	"time"

	"github.com/elastic/gosigar"
)

type SystemLoad struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

type CpuTimes struct {
	sigar.Cpu
	UserPercent   float64 `json:"user_p"`
	SystemPercent float64 `json:"system_p"`
}

type MemStat struct {
	sigar.Mem
	UsedPercent       float64 `json:"used_p"`
	ActualUsedPercent float64 `json:"actual_used_p"`
}

type SwapStat struct {
	sigar.Swap
	UsedPercent float64 `json:"used_p"`
}

type Process struct {
	Pid     int    `json:"pid"`
	Ppid    int    `json:"ppid"`
	Name    string `json:"name"`
	State   string `json:"state"`
	CmdLine string `json:"cmdline"`
	Mem     sigar.ProcMem
	Cpu     sigar.ProcTime
	ctime   time.Time
}

type FileSystemStat struct {
	sigar.FileSystemUsage
	DevName     string  `json:"device_name"`
	Mount       string  `json:"mount_point"`
	UsedPercent float64 `json:"used_p"`
	ctime       time.Time
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

	return &CpuTimes{Cpu: cpu}, nil

}

func GetCpuTimesList() ([]CpuTimes, error) {

	cpuList := sigar.CpuList{}
	err := cpuList.Get()
	if err != nil {
		return nil, err
	}

	cpuTimes := make([]CpuTimes, len(cpuList.List))

	for i, cpu := range cpuList.List {
		cpuTimes[i] = CpuTimes{Cpu: cpu}
	}

	return cpuTimes, nil
}

func GetMemory() (*MemStat, error) {

	mem := sigar.Mem{}
	err := mem.Get()
	if err != nil {
		return nil, err
	}

	return &MemStat{Mem: mem}, nil
}

func GetSwap() (*SwapStat, error) {

	swap := sigar.Swap{}
	err := swap.Get()
	if err != nil {
		return nil, err
	}

	return &SwapStat{Swap: swap}, nil

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
	args := sigar.ProcArgs{}

	err := state.Get(pid)
	if err != nil {
		return nil, fmt.Errorf("Error getting state info: %v", err)
	}

	err = mem.Get(pid)
	if err != nil {
		return nil, fmt.Errorf("Error getting mem info: %v", err)
	}

	err = cpu.Get(pid)
	if err != nil {
		return nil, fmt.Errorf("Error getting cpu info: %v", err)
	}

	err = args.Get(pid)
	if err != nil {
		return nil, fmt.Errorf("Error getting command line: %v", err)
	}

	cmdLine := strings.Join(args.List, " ")

	proc := Process{
		Pid:     pid,
		Ppid:    state.Ppid,
		Name:    state.Name,
		State:   getProcState(byte(state.State)),
		CmdLine: cmdLine,
		Mem:     mem,
		Cpu:     cpu,
	}
	proc.ctime = time.Now()

	return &proc, nil
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
		FileSystemUsage: stat,
		DevName:         fs.DevName,
		Mount:           fs.DirName,
	}

	return &filesystem, nil
}
