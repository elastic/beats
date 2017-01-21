// +build !integration
// +build darwin freebsd linux windows

package process

import (
	"os"
	"runtime"
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

	assert.True(t, (process.Ctime.Unix() <= time.Now().Unix()))

	switch runtime.GOOS {
	case "darwin", "linux", "freebsd":
		assert.True(t, len(process.Env) > 0, "empty environment")
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
	procStats := ProcStats{}

	ctime := time.Now()

	p2 := Process{
		Pid: 3545,
		Cpu: gosigar.ProcTime{
			User:  14794,
			Sys:   47,
			Total: 14841,
		},
		Ctime: ctime,
	}

	p1 := Process{
		Pid: 3545,
		Cpu: gosigar.ProcTime{
			User:  11345,
			Sys:   37,
			Total: 11382,
		},
		Ctime: ctime.Add(-1 * time.Second),
	}

	procStats.ProcsMap = make(ProcsMap)
	procStats.ProcsMap[p1.Pid] = &p1

	totalPercent := GetProcCpuPercentage(&p1, &p2)
	assert.Equal(t, totalPercent, 3.459)
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
