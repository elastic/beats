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

//go:build darwin || freebsd || linux || windows
// +build darwin freebsd linux windows

package process

import (
	"os"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/metric/system/cgroup"
	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/libbeat/opt"
)

// numCPU is the number of CPUs of the host
var numCPU = runtime.NumCPU()

func TestGetOne(t *testing.T) {
	testConfig := Stats{
		Procs:        []string{".*"},
		Hostfs:       resolve.NewTestResolver("/"),
		CPUTicks:     true,
		CacheCmdLine: true,
		EnvWhitelist: []string{".*"},
		IncludeTop: IncludeTopConfig{
			Enabled:  true,
			ByCPU:    4,
			ByMemory: 4,
		},
		EnableCgroups: false,
		CgroupOpts: cgroup.ReaderOptions{
			RootfsMountpoint:  resolve.NewTestResolver("/"),
			IgnoreRootCgroups: true,
		},
	}
	err := testConfig.Init()
	assert.NoError(t, err, "Init")

	procData, err := testConfig.GetOne(os.Getpid())
	assert.NoError(t, err, "GetOne")
	t.Logf("Proc: %s", procData.StringToPrint())
}

func TestGetProcess(t *testing.T) {
	stat, err := initTestResolver()
	assert.NoError(t, err, "Init()")
	process, err := stat.GetSelf()
	assert.NoError(t, err, "FetchPid")

	assert.True(t, (process.Pid.ValueOr(0) > 0))
	assert.True(t, (process.Ppid.ValueOr(0) >= 0))
	assert.True(t, (process.Pgid.ValueOr(0) >= 0))
	assert.True(t, (len(process.Name) > 0))
	assert.True(t, (len(process.Username) > 0))
	assert.NotEqual(t, "unknown", process.State)

	// Memory Checks
	assert.True(t, (process.Memory.Size.ValueOr(0) >= 0))
	assert.True(t, (process.Memory.Rss.Bytes.ValueOr(0) >= 0))
	assert.True(t, (process.Memory.Share.ValueOr(0) >= 0))

	// CPU Checks
	assert.True(t, (process.CPU.Total.Value.ValueOr(0) >= 0))
	assert.True(t, (process.CPU.User.Ticks.ValueOr(0) >= 0))
	assert.True(t, (process.CPU.System.Ticks.ValueOr(0) >= 0))

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

// See https://github.com/elastic/beats/issues/6620
func TestGetSelfPid(t *testing.T) {
	pid, err := GetSelfPid(resolve.NewTestResolver("/"))
	assert.NoError(t, err)
	assert.Equal(t, os.Getpid(), pid)
}

func TestMatchProcs(t *testing.T) {
	var procStats = Stats{}

	procStats.Procs = []string{".*"}
	err := procStats.Init()
	assert.NoError(t, err)

	assert.True(t, procStats.matchProcess("metricbeat"))

	procStats.Procs = []string{"metricbeat"}
	err = procStats.Init()
	assert.NoError(t, err)
	assert.False(t, procStats.matchProcess("burn"))

	// match no processes
	procStats.Procs = []string{"$^"}
	err = procStats.Init()
	assert.NoError(t, err)
	assert.False(t, procStats.matchProcess("burn"))
}

func TestProcMemPercentage(t *testing.T) {
	procStats := Stats{}

	p := ProcState{
		Pid: opt.IntWith(3456),
		Memory: ProcMemInfo{
			Rss:  MemBytePct{Bytes: opt.UintWith(1416)},
			Size: opt.UintWith(145164088),
		},
	}

	procStats.ProcsMap = make(ProcsMap)
	procStats.ProcsMap[p.Pid.ValueOr(0)] = p

	rssPercent := GetProcMemPercentage(p, 10000)
	assert.Equal(t, rssPercent.ValueOr(0), 0.1416)
}

func TestProcCpuPercentage(t *testing.T) {
	p1 := ProcState{
		CPU: ProcCPUInfo{
			User:   CPUTicks{Ticks: opt.UintWith(11345)},
			System: CPUTicks{Ticks: opt.UintWith(37)},
			Total: CPUTotal{
				Ticks: opt.UintWith(11382),
			},
		},
		SampleTime: time.Now(),
	}

	p2 := ProcState{
		CPU: ProcCPUInfo{
			User:   CPUTicks{Ticks: opt.UintWith(14794)},
			System: CPUTicks{Ticks: opt.UintWith(47)},
			Total: CPUTotal{
				Ticks: opt.UintWith(14841),
			},
		},
		SampleTime: p1.SampleTime.Add(time.Second),
	}

	totalPercentNormalized, totalPercent, totalValue := GetProcCPUPercentage(p1, p2)
	//GetProcCPUPercentage wil return a number that varies based on the host, due to NumCPU()
	// So "un-normalize" it, then re-normalized with a constant.
	cpu := float64(runtime.NumCPU())
	unNormalized := totalPercentNormalized * cpu
	normalizedTest := common.Round(unNormalized/48, common.DefaultDecimalPlacesCount)

	assert.EqualValues(t, 0.0721, normalizedTest)
	assert.EqualValues(t, 3.459, totalPercent)
	assert.EqualValues(t, 14841, totalValue)
}

// BenchmarkGetProcess runs a benchmark of the GetProcess method with caching
// of the command line and environment variables.
func BenchmarkGetProcess(b *testing.B) {
	stat, err := initTestResolver()
	if err != nil {
		b.Fatalf("Failed init: %s", err)
	}
	procs := make(map[int]common.MapStr, 1)
	pid := os.Getpid()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {

		process, err := stat.GetOne(pid)
		if err != nil {
			continue
		}

		procs[pid] = process
	}
}

func BenchmarkGetTop(b *testing.B) {
	stat, err := initTestResolver()
	if err != nil {
		b.Fatalf("Failed init: %s", err)
	}
	procs := make(map[int][]common.MapStr)

	for i := 0; i < b.N; i++ {
		list, _, err := stat.Get()
		if err != nil {
			b.Fatalf("error: %s", err)
		}
		procs[i] = list
	}
}

func TestIncludeTopProcesses(t *testing.T) {
	processes := []ProcState{
		{
			Pid: opt.IntWith(1),
			CPU: ProcCPUInfo{
				Total: CPUTotal{
					Pct: opt.FloatWith(10),
				},
			},
			Memory: ProcMemInfo{
				Rss: MemBytePct{
					Bytes: opt.UintWith(3000),
				},
			},
		},
		{
			Pid: opt.IntWith(2),
			CPU: ProcCPUInfo{
				Total: CPUTotal{
					Pct: opt.FloatWith(5),
				},
			},
			Memory: ProcMemInfo{
				Rss: MemBytePct{
					Bytes: opt.UintWith(4000),
				},
			},
		},
		{
			Pid: opt.IntWith(3),
			CPU: ProcCPUInfo{
				Total: CPUTotal{
					Pct: opt.FloatWith(7),
				},
			},
			Memory: ProcMemInfo{
				Rss: MemBytePct{
					Bytes: opt.UintWith(2000),
				},
			},
		},
		{
			Pid: opt.IntWith(4),
			CPU: ProcCPUInfo{
				Total: CPUTotal{
					Pct: opt.FloatWith(5),
				},
			},
			Memory: ProcMemInfo{
				Rss: MemBytePct{
					Bytes: opt.UintWith(8000),
				},
			},
		},
		{
			Pid: opt.IntWith(5),
			CPU: ProcCPUInfo{
				Total: CPUTotal{
					Pct: opt.FloatWith(12),
				},
			},
			Memory: ProcMemInfo{
				Rss: MemBytePct{
					Bytes: opt.UintWith(9000),
				},
			},
		},
		{
			Pid: opt.IntWith(6),
			CPU: ProcCPUInfo{
				Total: CPUTotal{
					Pct: opt.FloatWith(5),
				},
			},
			Memory: ProcMemInfo{
				Rss: MemBytePct{
					Bytes: opt.UintWith(7000),
				},
			},
		},
		{
			Pid: opt.IntWith(7),
			CPU: ProcCPUInfo{
				Total: CPUTotal{
					Pct: opt.FloatWith(80),
				},
			},
			Memory: ProcMemInfo{
				Rss: MemBytePct{
					Bytes: opt.UintWith(11000),
				},
			},
		},
		{
			Pid: opt.IntWith(8),
			CPU: ProcCPUInfo{
				Total: CPUTotal{
					Pct: opt.FloatWith(50),
				},
			},
			Memory: ProcMemInfo{
				Rss: MemBytePct{
					Bytes: opt.UintWith(13000),
				},
			},
		},
		{
			Pid: opt.IntWith(9),
			CPU: ProcCPUInfo{
				Total: CPUTotal{
					Pct: opt.FloatWith(15),
				},
			},
			Memory: ProcMemInfo{
				Rss: MemBytePct{
					Bytes: opt.UintWith(1000),
				},
			},
		},
		{
			Pid: opt.IntWith(10),
			CPU: ProcCPUInfo{
				Total: CPUTotal{
					Pct: opt.FloatWith(60),
				},
			},
			Memory: ProcMemInfo{
				Rss: MemBytePct{
					Bytes: opt.UintWith(500),
				},
			},
		},
	}

	tests := []struct {
		Name         string
		Cfg          IncludeTopConfig
		ExpectedPids []int
	}{
		{
			Name:         "top 2 processes by CPU",
			Cfg:          IncludeTopConfig{Enabled: true, ByCPU: 2},
			ExpectedPids: []int{7, 10},
		},
		{
			Name:         "top 4 processes by CPU",
			Cfg:          IncludeTopConfig{Enabled: true, ByCPU: 4},
			ExpectedPids: []int{7, 10, 8, 9},
		},
		{
			Name:         "top 2 processes by memory",
			Cfg:          IncludeTopConfig{Enabled: true, ByMemory: 2},
			ExpectedPids: []int{8, 7},
		},
		{
			Name:         "top 4 processes by memory",
			Cfg:          IncludeTopConfig{Enabled: true, ByMemory: 4},
			ExpectedPids: []int{8, 7, 5, 4},
		},
		{
			Name:         "top 2 processes by CPU + top 2 by memory",
			Cfg:          IncludeTopConfig{Enabled: true, ByCPU: 2, ByMemory: 2},
			ExpectedPids: []int{7, 10, 8},
		},
		{
			Name:         "top 4 processes by CPU + top 4 by memory",
			Cfg:          IncludeTopConfig{Enabled: true, ByCPU: 4, ByMemory: 4},
			ExpectedPids: []int{7, 10, 8, 9, 5, 4},
		},
		{
			Name:         "enabled false",
			Cfg:          IncludeTopConfig{Enabled: false, ByCPU: 4, ByMemory: 4},
			ExpectedPids: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			Name:         "enabled true but cpu & mem not configured",
			Cfg:          IncludeTopConfig{Enabled: true},
			ExpectedPids: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			Name:         "top 12 by cpu (out of 10)",
			Cfg:          IncludeTopConfig{Enabled: true, ByCPU: 12},
			ExpectedPids: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			Name:         "top 12 by cpu and top 14 memory (out of 10)",
			Cfg:          IncludeTopConfig{Enabled: true, ByCPU: 12, ByMemory: 14},
			ExpectedPids: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			Name:         "top 14 by cpu and top 12 memory (out of 10)",
			Cfg:          IncludeTopConfig{Enabled: true, ByCPU: 14, ByMemory: 12},
			ExpectedPids: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			Name:         "top 1 by cpu and top 3 memory",
			Cfg:          IncludeTopConfig{Enabled: true, ByCPU: 1, ByMemory: 3},
			ExpectedPids: []int{5, 7, 8},
		},
		{
			Name:         "top 3 by cpu and top 1 memory",
			Cfg:          IncludeTopConfig{Enabled: true, ByCPU: 3, ByMemory: 1},
			ExpectedPids: []int{7, 8, 10},
		},
	}

	for _, test := range tests {
		procStats := Stats{IncludeTop: test.Cfg}
		res := procStats.includeTopProcesses(processes)

		resPids := []int{}
		for _, p := range res {
			resPids = append(resPids, p.Pid.ValueOr(0))
		}
		sort.Ints(test.ExpectedPids)
		sort.Ints(resPids)
		assert.Equal(t, resPids, test.ExpectedPids, test.Name)
	}
}

func initTestResolver() (Stats, error) {
	logp.DevelopmentSetup()
	testConfig := Stats{
		Procs:        []string{".*"},
		Hostfs:       resolve.NewTestResolver("/"),
		CPUTicks:     true,
		CacheCmdLine: true,
		EnvWhitelist: []string{".*"},
		IncludeTop: IncludeTopConfig{
			Enabled:  true,
			ByCPU:    5,
			ByMemory: 5,
		},
		EnableCgroups: true,
		CgroupOpts: cgroup.ReaderOptions{
			RootfsMountpoint:  resolve.NewTestResolver("/"),
			IgnoreRootCgroups: true,
		},
	}
	err := testConfig.Init()
	return testConfig, err
}
