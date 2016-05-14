package gosigar_test

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	sigar "github.com/elastic/gosigar"
	"github.com/stretchr/testify/assert"
)

var procd string

func setUp(t testing.TB) {
	var err error
	procd, err = ioutil.TempDir("", "sigarTests")
	if err != nil {
		t.Fatal(err)
	}
	sigar.Procd = procd
}

func tearDown(t testing.TB) {
	sigar.Procd = "/proc"
	err := os.RemoveAll(procd)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLinuxProcState(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	var procNames = []string{
		"cron",
		"a very long process name",
	}

	for _, n := range procNames {
		func() {
			pid := rand.Int()
			pidDir := filepath.Join(procd, strconv.Itoa(pid))
			err := os.Mkdir(pidDir, 0755)
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(pidDir)
			pidStatFile := filepath.Join(pidDir, "stat")
			writePidStats(pid, n, pidStatFile)
			if err != nil {
				t.Fatal(err)
			}

			pidStatusFile := filepath.Join(pidDir, "status")
			uid := 123456789
			writePidStatus(n, pid, uid, pidStatusFile)
			if err != nil {
				t.Fatal(err)
			}

			state := sigar.ProcState{}
			if assert.NoError(t, state.Get(pid)) {
				assert.Equal(t, n, state.Name)
				assert.Equal(t, 2, state.Pgid)
				assert.Equal(t, strconv.Itoa(uid), state.Username)
			}
		}()
	}
}

func TestLinuxCPU(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	tests := []struct {
		stat string
		user uint64
	}{
		{"cpu 25 1 2 3 4 5 6 7", 25},
		// Ignore empty lines
		{"cpu ", 0},
	}

	statFile := procd + "/stat"
	for _, test := range tests {
		func() {
			statContents := []byte(test.stat)
			err := ioutil.WriteFile(statFile, statContents, 0644)
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(statFile)

			cpu := sigar.Cpu{}
			if assert.NoError(t, cpu.Get()) {
				assert.Equal(t, uint64(test.user), cpu.User, "cpu.User")
			}
		}()
	}
}

func TestLinuxCollectCpuStats(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	statFile := procd + "/stat"
	statContents := []byte("cpu 25 1 2 3 4 5 6 7")
	err := ioutil.WriteFile(statFile, statContents, 0644)
	if err != nil {
		t.Fatal(err)
	}

	concreteSigar := &sigar.ConcreteSigar{}
	cpuUsages, stop := concreteSigar.CollectCpuStats(500 * time.Millisecond)

	assert.Equal(t, sigar.Cpu{
		User:    uint64(25),
		Nice:    uint64(1),
		Sys:     uint64(2),
		Idle:    uint64(3),
		Wait:    uint64(4),
		Irq:     uint64(5),
		SoftIrq: uint64(6),
		Stolen:  uint64(7),
	}, <-cpuUsages)

	statContents = []byte("cpu 30 3 7 10 25 55 36 65")
	err = ioutil.WriteFile(statFile, statContents, 0644)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, sigar.Cpu{
		User:    uint64(5),
		Nice:    uint64(2),
		Sys:     uint64(5),
		Idle:    uint64(7),
		Wait:    uint64(21),
		Irq:     uint64(50),
		SoftIrq: uint64(30),
		Stolen:  uint64(58),
	}, <-cpuUsages)

	stop <- struct{}{}
}

func TestLinuxMemAndSwap(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	meminfoContents := `
MemTotal:         374256 kB
MemFree:          274460 kB
Buffers:            9764 kB
Cached:            38648 kB
SwapCached:            0 kB
Active:            33772 kB
Inactive:          31184 kB
Active(anon):      16572 kB
Inactive(anon):      552 kB
Active(file):      17200 kB
Inactive(file):    30632 kB
Unevictable:           0 kB
Mlocked:               0 kB
SwapTotal:        786428 kB
SwapFree:         786428 kB
Dirty:                 0 kB
Writeback:             0 kB
AnonPages:         16564 kB
Mapped:             6612 kB
Shmem:               584 kB
Slab:              19092 kB
SReclaimable:       9128 kB
SUnreclaim:         9964 kB
KernelStack:         672 kB
PageTables:         1864 kB
NFS_Unstable:          0 kB
Bounce:                0 kB
WritebackTmp:          0 kB
CommitLimit:      973556 kB
Committed_AS:      55880 kB
VmallocTotal:   34359738367 kB
VmallocUsed:       21428 kB
VmallocChunk:   34359713596 kB
HardwareCorrupted:     0 kB
AnonHugePages:         0 kB
HugePages_Total:       0
HugePages_Free:        0
HugePages_Rsvd:        0
HugePages_Surp:        0
Hugepagesize:       2048 kB
DirectMap4k:       59328 kB
DirectMap2M:      333824 kB
`

	meminfoFile := procd + "/meminfo"
	err := ioutil.WriteFile(meminfoFile, []byte(meminfoContents), 0444)
	if err != nil {
		t.Fatal(err)
	}

	mem := sigar.Mem{}
	if assert.NoError(t, mem.Get()) {
		assert.Equal(t, uint64(374256*1024), mem.Total)
		assert.Equal(t, uint64(274460*1024), mem.Free)
	}

	swap := sigar.Swap{}
	if assert.NoError(t, swap.Get()) {
		assert.Equal(t, uint64(786428*1024), swap.Total)
		assert.Equal(t, uint64(786428*1024), swap.Free)
	}
}

func writePidStats(pid int, procName string, path string) error {
	stats := "S 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 " +
		"20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 " +
		"35 36 37 38 39"

	statContents := []byte(fmt.Sprintf("%d (%s) %s", pid, procName, stats))
	return ioutil.WriteFile(path, statContents, 0644)
}

func writePidStatus(name string, pid int, uid int, pidStatusFile string) error {
	status := `
Name:   %s
State:  R (running)
Tgid:   5452
Pid:    %d
PPid:   743
TracerPid:      0
Uid:    %d     %d     %d    %d
Gid:    100     100     100     100
FDSize: 256
Groups: 100 14 16
VmPeak:     5004 kB
VmSize:     5004 kB
VmLck:         0 kB
VmHWM:       476 kB
VmRSS:       476 kB
RssAnon:             352 kB
RssFile:             120 kB
RssShmem:              4 kB
VmData:      156 kB
VmStk:        88 kB
VmExe:        68 kB
VmLib:      1412 kB
VmPTE:        20 kb
VmSwap:        0 kB
HugetlbPages:          0 kB
Threads:        1
SigQ:   0/28578
SigPnd: 0000000000000000
ShdPnd: 0000000000000000
SigBlk: 0000000000000000
SigIgn: 0000000000000000
SigCgt: 0000000000000000
CapInh: 00000000fffffeff
CapPrm: 0000000000000000
CapEff: 0000000000000000
CapBnd: ffffffffffffffff
Seccomp:        0
voluntary_ctxt_switches:        0
nonvoluntary_ctxt_switches:     1`

	statusContents := []byte(fmt.Sprintf(status, name, pid, uid))
	return ioutil.WriteFile(pidStatusFile, statusContents, 0644)
}
