// +build !integration
// +build darwin freebsd linux windows

package process

import (
	"os"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/gosigar"
	"github.com/stretchr/testify/assert"
)

func TestPids(t *testing.T) {
	pids, err := Pids()

	assert.NotNil(t, pids)
	assert.Nil(t, err)

	// Assuming at least 2 processes are running
	assert.True(t, (len(pids) > 1))
}

func TestGetProcess(t *testing.T) {
	process, err := newProcess(os.Getpid(), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = process.getDetails(nil); err != nil {
		t.Fatal(err)
	}

	assert.True(t, (process.Pid > 0))
	assert.True(t, (process.Ppid >= 0))
	assert.True(t, (process.Pgid >= 0))
	assert.True(t, (len(process.Name) > 0))
	assert.True(t, (len(process.Username) > 0))
	assert.NotEqual(t, "unknown", process.State)

	// Memory Checks
	assert.True(t, (process.Mem.Size >= 0))
	assert.True(t, (process.Mem.Resident >= 0))
	assert.True(t, (process.Mem.Share >= 0))

	// CPU Checks
	assert.True(t, (process.Cpu.StartTime > 0))
	assert.True(t, (process.Cpu.Total >= 0))
	assert.True(t, (process.Cpu.User >= 0))
	assert.True(t, (process.Cpu.Sys >= 0))

	assert.True(t, (process.SampleTime.Unix() <= time.Now().Unix()))

	switch runtime.GOOS {
	case "darwin", "linux", "freebsd":
		assert.True(t, len(process.Env) > 0, "empty environment")
	}

	switch runtime.GOOS {
	case "linux":
		assert.True(t, (len(process.Cwd) > 0))
	}
}

func TestProcState(t *testing.T) {
	assert.Equal(t, getProcState('R'), "running")
	assert.Equal(t, getProcState('S'), "sleeping")
	assert.Equal(t, getProcState('s'), "unknown")
	assert.Equal(t, getProcState('D'), "idle")
	assert.Equal(t, getProcState('T'), "stopped")
	assert.Equal(t, getProcState('Z'), "zombie")
}

func TestMatchProcs(t *testing.T) {
	var procStats = ProcStats{}

	procStats.Procs = []string{".*"}
	err := procStats.InitProcStats()
	assert.NoError(t, err)

	assert.True(t, procStats.MatchProcess("metricbeat"))

	procStats.Procs = []string{"metricbeat"}
	err = procStats.InitProcStats()
	assert.NoError(t, err)
	assert.False(t, procStats.MatchProcess("burn"))

	// match no processes
	procStats.Procs = []string{"$^"}
	err = procStats.InitProcStats()
	assert.NoError(t, err)
	assert.False(t, procStats.MatchProcess("burn"))
}

func TestProcMemPercentage(t *testing.T) {
	procStats := ProcStats{}

	p := Process{
		Pid: 3456,
		Mem: gosigar.ProcMem{
			Resident: 1416,
			Size:     145164088,
		},
	}

	procStats.ProcsMap = make(ProcsMap)
	procStats.ProcsMap[p.Pid] = &p

	rssPercent := GetProcMemPercentage(&p, 10000)
	assert.Equal(t, rssPercent, 0.1416)
}

func TestProcCpuPercentage(t *testing.T) {
	p1 := &Process{
		Cpu: gosigar.ProcTime{
			User:  11345,
			Sys:   37,
			Total: 11382,
		},
		SampleTime: time.Now(),
	}

	p2 := &Process{
		Cpu: gosigar.ProcTime{
			User:  14794,
			Sys:   47,
			Total: 14841,
		},
		SampleTime: p1.SampleTime.Add(time.Second),
	}

	NumCPU = 48
	defer func() { NumCPU = runtime.NumCPU() }()

	totalPercentNormalized, totalPercent := GetProcCpuPercentage(p1, p2)
	assert.EqualValues(t, 0.0721, totalPercentNormalized)
	assert.EqualValues(t, 3.459, totalPercent)
}

// BenchmarkGetProcess runs a benchmark of the GetProcess method with caching
// of the command line and environment variables.
func BenchmarkGetProcess(b *testing.B) {
	pids, err := Pids()
	if err != nil {
		b.Fatal(err)
	}
	nPids := len(pids)
	procs := make(ProcsMap, nPids)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pid := pids[i%nPids]

		var cmdline string
		var env common.MapStr
		if p := procs[pid]; p != nil {
			cmdline = p.CmdLine
			env = p.Env
		}

		process, err := newProcess(pid, cmdline, env)
		if err != nil {
			continue
		}
		err = process.getDetails(nil)
		assert.NoError(b, err)

		procs[pid] = process
	}
}

func TestIncludeTopProcesses(t *testing.T) {
	processes := []Process{
		{
			Pid:         1,
			cpuTotalPct: 10,
			Mem:         gosigar.ProcMem{Resident: 3000},
		},
		{
			Pid:         2,
			cpuTotalPct: 5,
			Mem:         gosigar.ProcMem{Resident: 4000},
		},
		{
			Pid:         3,
			cpuTotalPct: 7,
			Mem:         gosigar.ProcMem{Resident: 2000},
		},
		{
			Pid:         4,
			cpuTotalPct: 5,
			Mem:         gosigar.ProcMem{Resident: 8000},
		},
		{
			Pid:         5,
			cpuTotalPct: 12,
			Mem:         gosigar.ProcMem{Resident: 9000},
		},
		{
			Pid:         6,
			cpuTotalPct: 5,
			Mem:         gosigar.ProcMem{Resident: 7000},
		},
		{
			Pid:         7,
			cpuTotalPct: 80,
			Mem:         gosigar.ProcMem{Resident: 11000},
		},
		{
			Pid:         8,
			cpuTotalPct: 50,
			Mem:         gosigar.ProcMem{Resident: 13000},
		},
		{
			Pid:         9,
			cpuTotalPct: 15,
			Mem:         gosigar.ProcMem{Resident: 1000},
		},
		{
			Pid:         10,
			cpuTotalPct: 60,
			Mem:         gosigar.ProcMem{Resident: 500},
		},
	}

	tests := []struct {
		Name         string
		Cfg          includeTopConfig
		ExpectedPids []int
	}{
		{
			Name:         "top 2 processes by CPU",
			Cfg:          includeTopConfig{Enabled: true, ByCPU: 2},
			ExpectedPids: []int{7, 10},
		},
		{
			Name:         "top 4 processes by CPU",
			Cfg:          includeTopConfig{Enabled: true, ByCPU: 4},
			ExpectedPids: []int{7, 10, 8, 9},
		},
		{
			Name:         "top 2 processes by memory",
			Cfg:          includeTopConfig{Enabled: true, ByMemory: 2},
			ExpectedPids: []int{8, 7},
		},
		{
			Name:         "top 4 processes by memory",
			Cfg:          includeTopConfig{Enabled: true, ByMemory: 4},
			ExpectedPids: []int{8, 7, 5, 4},
		},
		{
			Name:         "top 2 processes by CPU + top 2 by memory",
			Cfg:          includeTopConfig{Enabled: true, ByCPU: 2, ByMemory: 2},
			ExpectedPids: []int{7, 10, 8},
		},
		{
			Name:         "top 4 processes by CPU + top 4 by memory",
			Cfg:          includeTopConfig{Enabled: true, ByCPU: 4, ByMemory: 4},
			ExpectedPids: []int{7, 10, 8, 9, 5, 4},
		},
		{
			Name:         "enabled false",
			Cfg:          includeTopConfig{Enabled: false, ByCPU: 4, ByMemory: 4},
			ExpectedPids: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			Name:         "enabled true but cpu & mem not configured",
			Cfg:          includeTopConfig{Enabled: true},
			ExpectedPids: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
	}

	for _, test := range tests {
		procStats := ProcStats{IncludeTop: test.Cfg}
		res := procStats.includeTopProcesses(processes)

		resPids := []int{}
		for _, p := range res {
			resPids = append(resPids, p.Pid)
		}
		sort.Ints(test.ExpectedPids)
		sort.Ints(resPids)
		assert.Equal(t, resPids, test.ExpectedPids, test.Name)
	}
}
