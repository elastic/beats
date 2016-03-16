// +build !integration

package system

import (
	"testing"
	"time"

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
	pids, err := Pids()

	assert.Nil(t, err)

	for _, pid := range pids {

		process, err := GetProcess(pid, "")

		if err != nil {
			continue
		}
		assert.NotNil(t, process)

		assert.True(t, (process.Pid > 0))
		assert.True(t, (process.Ppid >= 0))
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

		// it's enough to get valid data for a single process
		break
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

// BenchmarkGetProcess runs a benchmark of the GetProcess method with caching
// of the command line arguments enabled.
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
		if p := procs[pid]; p != nil {
			cmdline = p.CmdLine
		}

		process, err := GetProcess(pid, cmdline)
		if err != nil {
			continue
		}

		procs[pid] = process
	}
}
