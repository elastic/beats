package beater

import (
	"testing"
	"time"

	"github.com/elastic/gosigar"
	"github.com/stretchr/testify/assert"
)

func TestMatchProcs(t *testing.T) {

	var beat = Topbeat{}

	beat.procs = []string{".*"}
	assert.True(t, beat.MatchProcess("topbeat"))

	beat.procs = []string{"topbeat"}
	assert.False(t, beat.MatchProcess("burn"))

	// match no processes
	beat.procs = []string{"$^"}
	assert.False(t, beat.MatchProcess("burn"))
}

func TestMemPercentage(t *testing.T) {

	beat := Topbeat{}

	m := MemStat{
		Mem: sigar.Mem{
			Total: 7,
			Used:  5,
			Free:  2,
		},
	}
	beat.addMemPercentage(&m)
	assert.Equal(t, m.UsedPercent, 0.71)

	m = MemStat{
		Mem: sigar.Mem{Total: 0},
	}
	beat.addMemPercentage(&m)
	assert.Equal(t, m.UsedPercent, 0.0)
}

func TestActualMemPercentage(t *testing.T) {

	beat := Topbeat{}

	m := MemStat{
		Mem: sigar.Mem{
			Total:      7,
			ActualUsed: 5,
			ActualFree: 2,
		},
	}
	beat.addMemPercentage(&m)
	assert.Equal(t, m.ActualUsedPercent, 0.71)

	m = MemStat{
		Mem: sigar.Mem{
			Total: 0,
		},
	}
	beat.addMemPercentage(&m)
	assert.Equal(t, m.ActualUsedPercent, 0.0)
}

func TestCpuPercentage(t *testing.T) {

	beat := Topbeat{}

	cpu1 := CpuTimes{
		Cpu: sigar.Cpu{
			User:    10855311,
			Nice:    0,
			Sys:     2021040,
			Idle:    17657874,
			Wait:    0,
			Irq:     0,
			SoftIrq: 0,
			Stolen:  0,
		},
	}

	beat.addCpuPercentage(&cpu1)

	assert.Equal(t, cpu1.UserPercent, 0.0)
	assert.Equal(t, cpu1.SystemPercent, 0.0)

	cpu2 := CpuTimes{
		Cpu: sigar.Cpu{
			User:    10855693,
			Nice:    0,
			Sys:     2021058,
			Idle:    17657876,
			Wait:    0,
			Irq:     0,
			SoftIrq: 0,
			Stolen:  0,
		},
	}

	beat.addCpuPercentage(&cpu2)

	assert.Equal(t, cpu2.UserPercent, 0.95)
	assert.Equal(t, cpu2.SystemPercent, 0.04)
}

func TestProcMemPercentage(t *testing.T) {

	beat := Topbeat{}

	p := Process{
		Pid: 3456,
		Mem: sigar.ProcMem{
			Resident: 1416,
			Size:     145164088,
		},
	}

	beat.procsMap = make(ProcsMap)
	beat.procsMap[p.Pid] = &p

	rssPercent := beat.getProcMemPercentage(&p, 10000)
	assert.Equal(t, rssPercent, 0.14)
}

func TestProcCpuPercentage(t *testing.T) {

	beat := Topbeat{}

	ctime := time.Now()

	p2 := Process{
		Pid: 3545,
		Cpu: sigar.ProcTime{
			User:  14794,
			Sys:   47,
			Total: 14841,
		},
		ctime: ctime,
	}

	p1 := Process{
		Pid: 3545,
		Cpu: sigar.ProcTime{
			User:  11345,
			Sys:   37,
			Total: 11382,
		},
		ctime: ctime.Add(-1 * time.Second),
	}

	beat.procsMap = make(ProcsMap)
	beat.procsMap[p1.Pid] = &p1

	totalPercent := beat.getProcCpuPercentage(&p2)
	assert.Equal(t, totalPercent, 3.459)
}
