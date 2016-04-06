package gosigar_test

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/elastic/gosigar"
	"github.com/stretchr/testify/assert"
)

const invalidPid = 666666

func TestCpu(t *testing.T) {
	cpu := Cpu{}
	assert.NoError(t, cpu.Get())
}

func TestLoadAverage(t *testing.T) {
	avg := LoadAverage{}
	assert.NoError(t, avg.Get())
}

func TestUptime(t *testing.T) {
	skipWindows(t)
	uptime := Uptime{}
	if assert.NoError(t, uptime.Get()) {
		assert.True(t, uptime.Length > 0, "Uptime (%f) must be positive", uptime.Length)
	}
}

func TestMem(t *testing.T) {
	mem := Mem{}
	if assert.NoError(t, mem.Get()) {
		assert.True(t, mem.Total > 0, "mem.Total (%d) must be positive", mem.Total)
		assert.True(t, (mem.Used+mem.Free) <= mem.Total,
			"mem.Used (%d) + mem.Free (%d) must <= mem.Total (%d)",
			mem.Used, mem.Free, mem.Total)
	}
}

func TestSwap(t *testing.T) {
	swap := Swap{}
	if assert.NoError(t, swap.Get()) {
		assert.True(t, (swap.Used+swap.Free) <= swap.Total,
			"swap.Used (%d) + swap.Free (%d) must <= swap.Total (%d)",
			swap.Used, swap.Free, swap.Total)
	}
}

func TestCpuList(t *testing.T) {
	skipWindows(t)
	cpulist := CpuList{}
	if assert.NoError(t, cpulist.Get()) {
		numCore := len(cpulist.List)
		numCpu := runtime.NumCPU()
		assert.True(t, numCore >= numCpu, "Number of cores (%d) >= number of logical CPUs (%d)",
			numCore, numCpu)
	}
}

func TestFileSystemList(t *testing.T) {
	fslist := FileSystemList{}
	if assert.NoError(t, fslist.Get()) {
		assert.True(t, len(fslist.List) > 0)
	}
}

func TestFileSystemUsage(t *testing.T) {
	root := "/"
	if runtime.GOOS == "windows" {
		root = "C:\\"
	}
	fsusage := FileSystemUsage{}
	if assert.NoError(t, fsusage.Get(root)) {
		assert.True(t, fsusage.Total > 0)
	}
	assert.Error(t, fsusage.Get("T O T A L L Y B O G U S"))
}

func TestProcList(t *testing.T) {
	pids := ProcList{}
	if assert.NoError(t, pids.Get()) {
		assert.True(t, len(pids.List) > 2)
	}
}

func TestProcState(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}

	state := ProcState{}
	if assert.NoError(t, state.Get(os.Getppid())) {
		assert.Contains(t, []RunState{RunStateRun, RunStateSleep}, state.State)
		assert.Regexp(t, "go(.exe)?", state.Name)
		assert.Equal(t, u.Username, state.Username)
	}

	assert.Error(t, state.Get(invalidPid))
}

func TestProcMem(t *testing.T) {
	mem := ProcMem{}
	assert.NoError(t, mem.Get(os.Getppid()))

	assert.Error(t, mem.Get(invalidPid))
}

func TestProcTime(t *testing.T) {
	time := ProcTime{}
	assert.NoError(t, time.Get(os.Getppid()))

	assert.Error(t, time.Get(invalidPid))
}

func TestProcArgs(t *testing.T) {
	skipWindows(t)
	args := ProcArgs{}
	if assert.NoError(t, args.Get(os.Getppid())) {
		assert.True(t, len(args.List) >= 2)
	}
}

func TestProcExe(t *testing.T) {
	skipWindows(t)
	exe := ProcExe{}
	if assert.NoError(t, exe.Get(os.Getppid())) {
		assert.Regexp(t, "go(.exe)?", filepath.Base(exe.Name))
	}
}

func skipWindows(t testing.TB) {
	if runtime.GOOS == "windows" {
		t.Skipf("Skipping test on %s", runtime.GOOS)
	}
}
