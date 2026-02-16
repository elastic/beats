// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build linux

package cgv2

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/cgroup/cgcommon"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/cgroup/testhelpers"
)

const v2Path = "../testdata/docker/sys/fs/cgroup/system.slice/docker-1c8fa019edd4b9d4b2856f4932c55929c5c118c808ed5faee9a135ca6e84b039.scope"
const ubuntu = "../testdata/io_statfiles/ubuntu"
const ubuntu2 = "../testdata/io_statfiles/ubuntu2"

var testFileList = []string{
	"../testdata/docker.zip",
}

func TestMain(m *testing.M) {
	os.Exit(testhelpers.MainTestWrapper(m, testFileList))
}

func TestGetIO(t *testing.T) {
	ioTest := IOSubsystem{}
	err := ioTest.Get(v2Path, false)
	assert.NoError(t, err, "error in Get")

	goodStat := map[string]IOStat{
		"253:0": {
			Read:      IOMetric{Bytes: 1024, IOs: 1},
			Write:     IOMetric{Bytes: 4096, IOs: 1},
			Discarded: IOMetric{Bytes: 6, IOs: 8},
		},
		"8:0": {
			Read:      IOMetric{Bytes: 512, IOs: 100},
			Write:     IOMetric{Bytes: 4096, IOs: 1},
			Discarded: IOMetric{Bytes: 5, IOs: 23},
		},
	}

	assert.Equal(t, goodStat, ioTest.Stats)
}

func TestIostatFilesDuplicatedDeviceMetrics(t *testing.T) {
	ioTest := IOSubsystem{}
	err := ioTest.Get(ubuntu, false)
	assert.NoError(t, err, "error in Get")

	goodStat := map[string]IOStat{
		"7:7": {
			Read: IOMetric{
				Bytes: 556032,
				IOs:   78,
			},
			Write: IOMetric{
				Bytes: 0,
				IOs:   0,
			},
			Discarded: IOMetric{
				Bytes: 0,
				IOs:   0,
			},
		},
		"7:6": {
			Read: IOMetric{
				Bytes: 556032,
				IOs:   78,
			},
			Write: IOMetric{
				Bytes: 0,
				IOs:   0,
			},
			Discarded: IOMetric{
				Bytes: 0,
				IOs:   0,
			},
		},
		"7:5": {
			Read: IOMetric{
				Bytes: 556032,
				IOs:   78,
			},
			Write: IOMetric{
				Bytes: 0,
				IOs:   0,
			},
			Discarded: IOMetric{
				Bytes: 0,
				IOs:   0,
			},
		},
		"7:4": {
			Read: IOMetric{
				Bytes: 556032,
				IOs:   78,
			},
			Write: IOMetric{
				Bytes: 0,
				IOs:   0,
			},
			Discarded: IOMetric{
				Bytes: 0,
				IOs:   0,
			},
		},
		"7:3": {
			Read: IOMetric{
				Bytes: 21017600,
				IOs:   629,
			},
			Write: IOMetric{
				Bytes: 0,
				IOs:   0,
			},
			Discarded: IOMetric{
				Bytes: 0,
				IOs:   0,
			},
		},
	}

	assert.Equal(t, goodStat, ioTest.Stats)
}

func TestIOStatDeviceWithNoMetrics(t *testing.T) {
	ioTest := IOSubsystem{}
	err := ioTest.Get(ubuntu2, false)
	assert.NoError(t, err, "error in Get")

	goodStat := map[string]IOStat{
		"253:0": {
			Read: IOMetric{
				Bytes: 45931053056,
				IOs:   1078394,
			},
			Write: IOMetric{
				Bytes: 211814596608,
				IOs:   21426614,
			},
			Discarded: IOMetric{
				Bytes: 0,
				IOs:   0,
			},
		},
		"259:0": {
			Read: IOMetric{
				Bytes: 48963873792,
				IOs:   1315370,
			},
			Write: IOMetric{
				Bytes: 217588278272,
				IOs:   15358572,
			},
			Discarded: IOMetric{
				Bytes: 3222265856,
				IOs:   24,
			},
		},
	}
	assert.Equal(t, goodStat, ioTest.Stats)
}

func TestGetMem(t *testing.T) {
	mem := MemorySubsystem{}
	err := mem.Get(v2Path)
	assert.NoError(t, err, "error in GetV2")

	assert.Equal(t, uint64(3), mem.Mem.Events.High)
	assert.Equal(t, uint64(4), mem.Mem.Low.Bytes)
	assert.Equal(t, uint64(9125888), mem.Mem.Usage.Bytes)

	assert.Equal(t, uint64(17756400), mem.Stats.SlabReclaimable.Bytes)
	assert.Equal(t, uint64(12), mem.Stats.THPFaultAlloc)

	// Test memory pressure stall information
	expectedPressure := map[string]cgcommon.Pressure{
		"some": {
			Ten:          opt.Pct{Pct: 0.0},
			Sixty:        opt.Pct{Pct: 0.0},
			ThreeHundred: opt.Pct{Pct: 0.0},
			Total:        opt.UintWith(0),
		},
		"full": {
			Ten:          opt.Pct{Pct: 0.0},
			Sixty:        opt.Pct{Pct: 0.0},
			ThreeHundred: opt.Pct{Pct: 0.0},
			Total:        opt.UintWith(0),
		},
	}
	assert.Equal(t, expectedPressure, mem.Pressure)
}

func TestGetMemPressure(t *testing.T) {
	// Create a temp directory with memory.pressure file containing non-zero values
	tempDir := t.TempDir()

	// Create a memory.pressure file with meaningful values
	pressureContent := `some avg10=1.50 avg60=2.30 avg300=0.75 total=123456
full avg10=0.80 avg60=1.20 avg300=0.40 total=78901
`
	writeFile(t, tempDir+"/memory.pressure", pressureContent)
	writeFile(t, tempDir+"/memory.stat", "anon 0\n")

	mem := MemorySubsystem{}
	err := mem.Get(tempDir)
	assert.NoError(t, err, "error in Get")

	expectedPressure := map[string]cgcommon.Pressure{
		"some": {
			Ten:          opt.Pct{Pct: 1.50},
			Sixty:        opt.Pct{Pct: 2.30},
			ThreeHundred: opt.Pct{Pct: 0.75},
			Total:        opt.UintWith(123456),
		},
		"full": {
			Ten:          opt.Pct{Pct: 0.80},
			Sixty:        opt.Pct{Pct: 1.20},
			ThreeHundred: opt.Pct{Pct: 0.40},
			Total:        opt.UintWith(78901),
		},
	}
	assert.Equal(t, expectedPressure, mem.Pressure)
}

func TestGetMemNoPressure(t *testing.T) {
	// Test that memory subsystem works when memory.pressure doesn't exist
	tempDir := t.TempDir()

	// Create minimal required memory files but NOT memory.pressure
	writeFile(t, tempDir+"/memory.stat", "anon 0\n")

	mem := MemorySubsystem{}
	err := mem.Get(tempDir)
	assert.NoError(t, err, "error in Get - should not fail if memory.pressure is missing")
	assert.Empty(t, mem.Pressure, "Pressure should be empty when memory.pressure file doesn't exist")
}

func TestGetCPU(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		expected CPUSubsystem
	}{
		{
			name: "v2 path with pressure",
			setup: func(*testing.T) string {
				return v2Path
			},
			expected: CPUSubsystem{
				ID:   "",
				Path: "",
				Pressure: map[string]cgcommon.Pressure{
					"some": {
						Ten:          opt.Pct{Pct: 4.30},
						Sixty:        opt.Pct{Pct: 3.20},
						ThreeHundred: opt.Pct{Pct: 1.11},
						Total:        opt.UintWith(1676316),
					},
				},
				Stats: CPUStats{
					Usage: cgcommon.CPUUsage{
						NS: 26772130245,
					},
					User: cgcommon.CPUUsage{
						NS: 20979069928,
					},
					System: cgcommon.CPUUsage{
						NS: 5793060316,
					},
					Periods: opt.UintWith(1),
					Throttled: ThrottledField{
						Periods: opt.UintWith(4),
						Us:      opt.UintWith(10),
					},
				},
			},
		},
		{
			name: "empty directory",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			expected: CPUSubsystem{
				ID:       "",
				Path:     "",
				Pressure: map[string]cgcommon.Pressure{},
				Stats:    CPUStats{},
			},
		},
		{
			name: "cpu.stat only",
			setup: func(t *testing.T) string {
				b, err := os.ReadFile(filepath.Join(v2Path, "cpu.stat"))
				require.NoError(t, err)
				dir := t.TempDir()
				err = os.WriteFile(filepath.Join(dir, "cpu.stat"), b, 0644)
				require.NoError(t, err)
				return dir
			},
			expected: CPUSubsystem{
				ID:       "",
				Path:     "",
				Pressure: map[string]cgcommon.Pressure{},
				Stats: CPUStats{
					Usage: cgcommon.CPUUsage{
						NS: 26772130245,
					},
					User: cgcommon.CPUUsage{
						NS: 20979069928,
					},
					System: cgcommon.CPUUsage{
						NS: 5793060316,
					},
					Periods: opt.UintWith(1),
					Throttled: ThrottledField{
						Periods: opt.UintWith(4),
						Us:      opt.UintWith(10),
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cpu := CPUSubsystem{}
			err := cpu.Get(test.setup(t))
			require.NoError(t, err, "error in Get")
			assert.EqualValues(t, test.expected, cpu)
		})
	}
}

func TestFillStatStructZswap(t *testing.T) {
	// Based on /sys/fs/cgroup/user.slice/memory.stat (kernel 6.12.67-2).
	// Zero values changed to non-zero, ensuring complete field coverage.
	const statContent = `anon 4740431872
file 10583662592
kernel 2101587968
kernel_stack 18071552
pagetables 52617216
sec_pagetables 8192
percpu 947744
sock 12288
vmalloc 225280
shmem 2948644864
zswap 125829120
zswapped 367001600
file_mapped 862699520
file_dirty 962560
file_writeback 131072
swapcached 2097152
anon_thp 922746880
file_thp 465567744
shmem_thp 2321547264
inactive_anon 177168384
active_anon 4563263488
inactive_file 4908941312
active_file 3291152384
unevictable 2434473984
slab_reclaimable 1990437352
slab_unreclaimable 36692632
slab 2027129984
workingset_refault_anon 1500
workingset_refault_file 8500
workingset_activate_anon 800
workingset_activate_file 4200
workingset_restore_anon 300
workingset_restore_file 1800
workingset_nodereclaim 5248
pgdemote_kswapd 5000
pgdemote_direct 1000
pgdemote_khugepaged 200
pgpromote_success 3500
pgscan 911220
pgsteal 911162
pgscan_kswapd 860000
pgscan_direct 46220
pgscan_khugepaged 5000
pgsteal_kswapd 865000
pgsteal_direct 43162
pgsteal_khugepaged 3000
pgfault 35936410
pgmajfault 4088
pgrefill 61074
pgactivate 85000
pgdeactivate 45000
pglazyfree 565444
pglazyfreed 180000
swpin_zero 500
swpout_zero 800
zswpin 450000
zswpout 680000
zswpwb 15000
thp_fault_alloc 11356
thp_collapse_alloc 519
thp_swpout 350
thp_swpout_fallback 150
numa_pages_migrated 8500
numa_pte_updates 12000
numa_hint_faults 4500`
	tmpDir := t.TempDir()
	writeFile(t, filepath.Join(tmpDir, "memory.stat"), statContent)

	stats, err := fillStatStruct(tmpDir)
	require.NoError(t, err)

	expected := MemoryStat{
		Anon:                   opt.Bytes{Bytes: 4740431872},
		File:                   opt.Bytes{Bytes: 10583662592},
		KernelStack:            opt.Bytes{Bytes: 18071552},
		Pagetables:             opt.Bytes{Bytes: 52617216},
		PerCPU:                 opt.Bytes{Bytes: 947744},
		Sock:                   opt.Bytes{Bytes: 12288},
		Shmem:                  opt.Bytes{Bytes: 2948644864},
		FileMapped:             opt.Bytes{Bytes: 862699520},
		FileDirty:              opt.Bytes{Bytes: 962560},
		FileWriteback:          opt.Bytes{Bytes: 131072},
		SwapCached:             opt.Bytes{Bytes: 2097152},
		AnonTHP:                opt.Bytes{Bytes: 922746880},
		FileTHP:                opt.Bytes{Bytes: 465567744},
		ShmemTHP:               opt.Bytes{Bytes: 2321547264},
		InactiveAnon:           opt.Bytes{Bytes: 177168384},
		ActiveAnon:             opt.Bytes{Bytes: 4563263488},
		InactiveFile:           opt.Bytes{Bytes: 4908941312},
		ActiveFile:             opt.Bytes{Bytes: 3291152384},
		Unevictable:            opt.Bytes{Bytes: 2434473984},
		SlabReclaimable:        opt.Bytes{Bytes: 1990437352},
		SlabUnreclaimable:      opt.Bytes{Bytes: 36692632},
		Slab:                   opt.Bytes{Bytes: 2027129984},
		WorkingSetRefaultAnon:  1500,
		WorkingSetRefaultFile:  8500,
		WorkingSetActivateAnon: 800,
		WorkingSetActivateFile: 4200,
		WorkingSetRestoreAnon:  300,
		WorkingSetRestoreFile:  1800,
		WorkingSetNodeReclaim:  5248,
		PageFaults:             35936410,
		MajorPageFaults:        4088,
		PageRefill:             61074,
		PageScan:               911220,
		PageSteal:              911162,
		PageActivate:           85000,
		PageDeactivate:         45000,
		PageLazyFree:           565444,
		PageLazyFreed:          180000,
		THPFaultAlloc:          11356,
		THPCollapseAlloc:       519,
		Zswap:                  opt.Bytes{Bytes: 125829120},
		Zswapped:               opt.Bytes{Bytes: 367001600},
		Zswpin:                 450000,
		Zswpout:                680000,
		Zswpwb:                 15000,
	}
	assert.Equal(t, expected, stats)
}

func writeFile(t testing.TB, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}
