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

func TestLinuxMemAndSwapKernel_3_14(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	meminfoContents := `
MemTotal:         500184 kB
MemFree:           31360 kB
MemAvailable:     414168 kB
Buffers:           28740 kB
Cached:           325408 kB
SwapCached:          264 kB
Active:           195476 kB
Inactive:         198612 kB
Active(anon):      14920 kB
Inactive(anon):    27268 kB
Active(file):     180556 kB
Inactive(file):   171344 kB
Unevictable:           0 kB
Mlocked:               0 kB
SwapTotal:        524284 kB
SwapFree:         520352 kB
Dirty:                 0 kB
Writeback:             0 kB
AnonPages:         39772 kB
Mapped:            24132 kB
Shmem:              2236 kB
Slab:              57988 kB
SReclaimable:      43524 kB
SUnreclaim:        14464 kB
KernelStack:        2464 kB
PageTables:         3096 kB
NFS_Unstable:          0 kB
Bounce:                0 kB
WritebackTmp:          0 kB
CommitLimit:      774376 kB
Committed_AS:     490916 kB
VmallocTotal:   34359738367 kB
VmallocUsed:           0 kB
VmallocChunk:          0 kB
HardwareCorrupted:     0 kB
AnonHugePages:         0 kB
CmaTotal:              0 kB
CmaFree:               0 kB
HugePages_Total:       0
HugePages_Free:        0
HugePages_Rsvd:        0
HugePages_Surp:        0
Hugepagesize:       2048 kB
DirectMap4k:       63424 kB
DirectMap2M:      460800 kB
`

	meminfoFile := procd + "/meminfo"
	err := ioutil.WriteFile(meminfoFile, []byte(meminfoContents), 0444)
	if err != nil {
		t.Fatal(err)
	}

	mem := sigar.Mem{}
	if assert.NoError(t, mem.Get()) {
		assert.Equal(t, uint64(500184*1024), mem.Total)
		assert.Equal(t, uint64(31360*1024), mem.Free)
		assert.Equal(t, uint64(414168*1024), mem.ActualFree)
	}

	swap := sigar.Swap{}
	if assert.NoError(t, swap.Get()) {
		assert.Equal(t, uint64(524284*1024), swap.Total)
		assert.Equal(t, uint64(520352*1024), swap.Free)
	}
}

func TestLinuxMemAndSwapMissingMemTotal(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	meminfoContents := `
MemFree:           31360 kB
MemAvailable:     414168 kB
Buffers:           28740 kB
Cached:           325408 kB
SwapCached:          264 kB
Active:           195476 kB
Inactive:         198612 kB
Active(anon):      14920 kB
Inactive(anon):    27268 kB
Active(file):     180556 kB
Inactive(file):   171344 kB
Unevictable:           0 kB
Mlocked:               0 kB
SwapTotal:        524284 kB
SwapFree:         520352 kB
Dirty:                 0 kB
Writeback:             0 kB
AnonPages:         39772 kB
Mapped:            24132 kB
Shmem:              2236 kB
Slab:              57988 kB
SReclaimable:      43524 kB
SUnreclaim:        14464 kB
KernelStack:        2464 kB
PageTables:         3096 kB
NFS_Unstable:          0 kB
Bounce:                0 kB
WritebackTmp:          0 kB
CommitLimit:      774376 kB
Committed_AS:     490916 kB
VmallocTotal:   34359738367 kB
VmallocUsed:           0 kB
VmallocChunk:          0 kB
HardwareCorrupted:     0 kB
AnonHugePages:         0 kB
CmaTotal:              0 kB
CmaFree:               0 kB
HugePages_Total:       0
HugePages_Free:        0
HugePages_Rsvd:        0
HugePages_Surp:        0
Hugepagesize:       2048 kB
DirectMap4k:       63424 kB
DirectMap2M:      460800 kB
`

	meminfoFile := procd + "/meminfo"
	err := ioutil.WriteFile(meminfoFile, []byte(meminfoContents), 0444)
	if err != nil {
		t.Fatal(err)
	}

	mem := sigar.Mem{}
	if assert.NoError(t, mem.Get()) {
		assert.Equal(t, uint64(0), mem.Total)
		assert.Equal(t, uint64(31360*1024), mem.Free)
		assert.Equal(t, uint64(414168*1024), mem.ActualFree)
	}

	swap := sigar.Swap{}
	if assert.NoError(t, swap.Get()) {
		assert.Equal(t, uint64(524284*1024), swap.Total)
		assert.Equal(t, uint64(520352*1024), swap.Free)
	}
}

func TestLinuxMemAndSwapKernel_3_14_memavailable_zero(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	meminfoContents := `
MemTotal:       148535680 kB
MemFree:          417356 kB
MemAvailable:          0 kB
Buffers:            1728 kB
Cached:           129928 kB
SwapCached:         8208 kB
Active:         141088676 kB
Inactive:        5568132 kB
Active(anon):   141076780 kB
Inactive(anon):  5556936 kB
Active(file):      11896 kB
Inactive(file):    11196 kB
Unevictable:        3648 kB
Mlocked:            3648 kB
SwapTotal:       4882428 kB
SwapFree:              0 kB
Dirty:               808 kB
Writeback:           220 kB
AnonPages:      146521272 kB
Mapped:            41384 kB
Shmem:            105864 kB
Slab:             522648 kB
SReclaimable:     233508 kB
SUnreclaim:       289140 kB
KernelStack:       85024 kB
PageTables:       368760 kB
NFS_Unstable:          0 kB
Bounce:                0 kB
WritebackTmp:          0 kB
CommitLimit:    79150268 kB
Committed_AS:   272491684 kB
VmallocTotal:   34359738367 kB
VmallocUsed:           0 kB
VmallocChunk:          0 kB
HardwareCorrupted:     0 kB
AnonHugePages:  78061568 kB
ShmemHugePages:        0 kB
ShmemPmdMapped:        0 kB
CmaTotal:              0 kB
CmaFree:               0 kB
HugePages_Total:       0
HugePages_Free:        0
HugePages_Rsvd:        0
HugePages_Surp:        0
Hugepagesize:       2048 kB
DirectMap4k:      124388 kB
DirectMap2M:     5105664 kB
DirectMap1G:    147849216 kB
`

	meminfoFile := procd + "/meminfo"
	err := ioutil.WriteFile(meminfoFile, []byte(meminfoContents), 0444)
	if err != nil {
		t.Fatal(err)
	}

	mem := sigar.Mem{}
	if assert.NoError(t, mem.Get()) {
		assert.Equal(t, uint64(148535680*1024), mem.Total)
		assert.Equal(t, uint64(417356*1024), mem.Free)
		assert.Equal(t, uint64(0), mem.ActualFree)
	}

	swap := sigar.Swap{}
	if assert.NoError(t, swap.Get()) {
		assert.Equal(t, uint64(4882428*1024), swap.Total)
		assert.Equal(t, uint64(0), swap.Free)
	}

}

func TestFDUsage(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	// There is no Uint63 until  2.0
	open := uint64(rand.Uint32())
	unused := uint64(rand.Uint32())
	max := uint64(rand.Uint32())
	fileNRContents := fmt.Sprintf("%d    %d       %d", open, unused, max)

	fileNRPath := procd + "/sys/fs"
	os.MkdirAll(fileNRPath, 0755)
	fileNRFile := fileNRPath + "/file-nr"
	err := ioutil.WriteFile(fileNRFile, []byte(fileNRContents), 0444)
	if err != nil {
		t.Fatal(err)
	}

	fd := sigar.FDUsage{}
	if assert.NoError(t, fd.Get()) {
		assert.Equal(t, open, fd.Open)
		assert.Equal(t, unused, fd.Unused)
		assert.Equal(t, max, fd.Max)
	}
}

func TestProcFDUsage(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	pid := rand.Intn(32768)
	pidDir := fmt.Sprintf("%s/%d", procd, pid)
	err := os.Mkdir(pidDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	soft := uint64(rand.Uint32())
	// subtract to prevent the posibility of overflow
	if soft != 0 {
		soft -= 1
	}
	// max sure hard is always bigger than soft
	hard := soft + uint64(rand.Uint32())

	limitsContents := `Limit                     Soft Limit           Hard Limit           Units
Max cpu time              unlimited            unlimited            seconds
Max file size             unlimited            unlimited            bytes
Max data size             unlimited            unlimited            bytes
Max stack size            8388608              unlimited            bytes
Max core file size        0                    unlimited            bytes
Max resident set          unlimited            unlimited            bytes
Max processes             29875                29875                processes
Max open files            %d                 %d                 files
Max locked memory         65536                65536                bytes
Max address space         unlimited            unlimited            bytes
Max file locks            unlimited            unlimited            locks
Max pending signals       29875                29875                signals
Max msgqueue size         819200               819200               bytes
Max nice priority         0                    0
Max realtime priority     0                    0
Max realtime timeout      unlimited            unlimited            us
`

	limitsContents = fmt.Sprintf(limitsContents, soft, hard)

	limitsFile := pidDir + "/limits"
	err = ioutil.WriteFile(limitsFile, []byte(limitsContents), 0444)
	if err != nil {
		t.Fatal(err)
	}
	open := rand.Intn(32768)
	if err = writeFDs(pid, open); err != nil {
		t.Fatal(err)
	}

	procFD := sigar.ProcFDUsage{}
	if assert.NoError(t, procFD.Get(pid)) {
		assert.Equal(t, uint64(open), procFD.Open)
		assert.Equal(t, soft, procFD.SoftLimit)
		assert.Equal(t, hard, procFD.HardLimit)
	}
}

func writeFDs(pid int, count int) error {
	fdDir := fmt.Sprintf("%s/%d/fd", procd, pid)
	err := os.Mkdir(fdDir, 0755)
	if err != nil {
		return err
	}

	for i := 0; i < count; i++ {
		fdPath := fmt.Sprintf("%s/%d", fdDir, i)
		f, err := os.Create(fdPath)
		if err != nil {
			return err
		}
		f.Close()
	}
	return nil
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
