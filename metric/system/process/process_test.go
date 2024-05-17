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

package process

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-system-metrics/metric"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/cgroup"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

func TestProcessEvent(t *testing.T) {
	proc := ProcState{Args: []string{"-b", "-c"},
		Name:     "test",
		Username: "user",
		Memory:   ProcMemInfo{Rss: MemBytePct{Pct: opt.FloatWith(4.5)}},
	}

	root := processRootEvent(&proc)

	require.Empty(t, proc.Name)
	require.Empty(t, proc.Username)
	require.Empty(t, proc.Args)

	require.NotNil(t, root["process"].(map[string]interface{})["memory"])
}

// BenchmarkGetProcess runs a benchmark of the GetProcess method with caching
// of the command line and environment variables.
func BenchmarkGetProcess(b *testing.B) {
	stat, err := initTestResolver()
	if err != nil {
		b.Fatalf("Failed init: %s", err)
	}
	procs := make(map[int]mapstr.M, 1)
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
	procs := make(map[int][]mapstr.M)

	for i := 0; i < b.N; i++ {
		list, _, err := stat.Get()
		if err != nil {
			b.Fatalf("error: %s", err)
		}
		procs[i] = list
	}
}

func TestGetState(t *testing.T) {
	wantRunning := Running
	wantSleepng := Sleeping
	pid := os.Getpid()
	hostfs := resolve.NewTestResolver("/")

	var got PidState
	var err error
	test := func() bool {
		// Getpid is really the only way to test this in a cross-platform way
		got, err = GetPIDState(hostfs, pid)
		if err != nil {
			return false
		}

		return wantSleepng == got || wantRunning == got
	}

	assert.Eventuallyf(t, test,
		time.Second*5, 50*time.Millisecond,
		"got process state %q. Last error: %v", got, err)
}

func TestGetOneRoot(t *testing.T) {
	testConfig := Stats{
		Procs:        []string{".*"},
		Hostfs:       resolve.NewTestResolver("/"),
		CPUTicks:     false,
		CacheCmdLine: true,
		EnvWhitelist: []string{".*"},
		IncludeTop: IncludeTopConfig{
			Enabled:  true,
			ByCPU:    4,
			ByMemory: 0,
		},
		EnableCgroups: false,
		CgroupOpts: cgroup.ReaderOptions{
			RootfsMountpoint:  resolve.NewTestResolver("/"),
			IgnoreRootCgroups: true,
		},
	}
	err := testConfig.Init()
	assert.NoError(t, err, "Init")

	evt, rootEvt, err := testConfig.GetOneRootEvent(os.Getpid())
	require.NoError(t, err)

	require.NotEmpty(t, rootEvt["process"].(map[string]interface{})["pid"])

	require.NotEmpty(t, evt["cpu"])
}

func TestGetOne(t *testing.T) {
	testConfig := Stats{
		Procs:        []string{".*"},
		Hostfs:       resolve.NewTestResolver("/"),
		CPUTicks:     false,
		CacheCmdLine: true,
		EnvWhitelist: []string{".*"},
		IncludeTop: IncludeTopConfig{
			Enabled:  true,
			ByCPU:    4,
			ByMemory: 0,
		},
		EnableCgroups: false,
		CgroupOpts: cgroup.ReaderOptions{
			RootfsMountpoint:  resolve.NewTestResolver("/"),
			IgnoreRootCgroups: true,
		},
	}
	err := testConfig.Init()
	assert.NoError(t, err, "Init")

	_, _, err = testConfig.Get()
	assert.NoError(t, err, "GetOne")

	time.Sleep(time.Second * 2)

	procData, _, err := testConfig.Get()
	assert.NoError(t, err, "GetOne")

	t.Logf("Proc: %s", procData[0].StringToPrint())
}

func TestNetworkFetch(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Network data only available on linux")
	}
	testConfig := Stats{
		Procs:         []string{".*"},
		Hostfs:        resolve.NewTestResolver("/"),
		CPUTicks:      false,
		EnableCgroups: false,
		EnableNetwork: true,
	}

	err := testConfig.Init()
	require.NoError(t, err)

	data, err := testConfig.GetOne(os.Getpid())
	require.NoError(t, err)
	networkData, ok := data["network"]
	require.True(t, ok, "network data not found")
	require.NotEmpty(t, networkData)
}

func TestNetworkFilter(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Network data only available on linux")
	}
	testConfig := Stats{
		Hostfs:         resolve.NewTestResolver("/"),
		EnableNetwork:  true,
		NetworkMetrics: []string{"Forwarding"},
	}

	err := testConfig.Init()
	require.NoError(t, err)

	data, err := testConfig.GetOne(os.Getpid())
	require.NoError(t, err)

	_, exists := data.GetValue("network.ip.Forwarding")
	require.NoError(t, exists, "filter did not preserve key")
	ipMetrics, _ := data.GetValue("network.ip")
	require.Equal(t, 1, len(ipMetrics.(map[string]interface{})))
}

func TestFilter(t *testing.T) {
	// The logic itself is os-independent, so we'll only test this on the platform least likely to have CI issues
	if runtime.GOOS != "linux" {
		t.Skip("Run on Linux only")
	}
	testConfig := Stats{
		Procs:  []string{".*"},
		Hostfs: resolve.NewTestResolver("/"),
		IncludeTop: IncludeTopConfig{
			Enabled:  true,
			ByCPU:    1,
			ByMemory: 1,
		},
	}
	err := testConfig.Init()
	assert.NoError(t, err, "Init")

	procData, _, err := testConfig.Get()
	assert.NoError(t, err, "GetOne")
	// the total count of processes can either be one or two,
	// depending on if the highest-mem-usage process and
	// highest-cpu-usage process are the same.
	assert.GreaterOrEqual(t, len(procData), 1)

	testZero := Stats{
		Procs:  []string{".*"},
		Hostfs: resolve.NewTestResolver("/"),
		IncludeTop: IncludeTopConfig{
			Enabled:  true,
			ByCPU:    0,
			ByMemory: 1,
		},
	}

	err = testZero.Init()
	assert.NoError(t, err, "Init")

	oneData, _, err := testZero.Get()
	assert.NoError(t, err, "GetOne with one event")
	assert.Equal(t, 1, len(oneData))
}

func TestProcessList(t *testing.T) {
	plist, err := ListStates(resolve.NewTestResolver("/"))
	assert.NoError(t, err, "ListStates")

	for _, proc := range plist {
		assert.NotEmpty(t, proc.State)
		assert.True(t, proc.Pid.Exists())
	}
}

func TestSelfPersist(t *testing.T) {
	stat, err := initTestResolver()
	require.NoError(t, err, "Init()")
	first, err := stat.GetSelf()
	require.NoError(t, err, "First GetSelf()")

	// The first process fetch shouldn't have percentages, since we don't have >1 procs to compare
	assert.False(t, first.CPU.Total.Pct.Exists(), "total.pct should not exist")
	// Create a proper time delay so the CPU percentage delta calculations don't fail
	time.Sleep(time.Millisecond * 5)
	second, err := stat.GetSelf()
	require.NoError(t, err, "Second GetSelf()")

	// now it should exist
	assert.True(t, second.CPU.Total.Pct.Exists(), "total.pct should exist")
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
	assert.True(t, process.Memory.Size.Exists())
	assert.True(t, (process.Memory.Rss.Bytes.ValueOr(0) > 0))
	if runtime.GOOS == "linux" {
		assert.True(t, process.Memory.Share.Exists())
	}

	// CPU Checks
	assert.True(t, (process.CPU.Total.Value.ValueOr(0) >= 0))
	assert.True(t, process.CPU.User.Ticks.Exists())
	assert.True(t, process.CPU.System.Ticks.Exists())

	assert.True(t, (process.SampleTime.Unix() <= time.Now().Unix()))

	switch runtime.GOOS {
	case "darwin", "linux", "freebsd":
		assert.True(t, len(process.Env) > 0, "empty environment")
	}

	switch runtime.GOOS {
	case "linux":
		assert.True(t, (len(process.Cwd) > 0))
	}

	assert.NotEmptyf(t, process.Cmdline, "cmdLine must be present")
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

	procStats.ProcsMap = NewProcsTrack()
	procStats.ProcsMap.SetPid(p.Pid.ValueOr(0), p)

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

	newState := GetProcCPUPercentage(p1, p2)
	// GetProcCPUPercentage wil return a number that varies based on the host, due to NumCPU()
	// So "un-normalize" it, then re-normalized with a constant.
	cpu := float64(runtime.NumCPU())
	unNormalized := newState.CPU.Total.Norm.Pct.ValueOr(0) * cpu
	normalizedTest := metric.Round(unNormalized / 48)

	assert.EqualValues(t, 0.0721, normalizedTest)
	assert.EqualValues(t, 3.459, newState.CPU.Total.Pct.ValueOr(0))
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

// runThreads run the threads binary for the current GOOS.
//
//go:generate docker run --rm -v ./testdata:/app --entrypoint g++ docker.elastic.co/beats-dev/golang-crossbuild:1.21.0-main -pthread -std=c++11 -o /app/threads /app/threads.cpp
//go:generate docker run --rm -v ./testdata:/app --entrypoint o64-clang++ docker.elastic.co/beats-dev/golang-crossbuild:1.21.0-darwin -pthread -std=c++11 -o /app/threads-darwin /app/threads.cpp
//go:generate docker run --rm -v ./testdata:/app --entrypoint x86_64-w64-mingw32-g++-posix docker.elastic.co/beats-dev/golang-crossbuild:1.21.0-main -pthread -std=c++11 -o /app/threads.exe /app/threads.cpp
func runThreads(t *testing.T) *exec.Cmd { //nolint: deadcode,structcheck,unused // needed by other platforms
	t.Helper()

	supportedPlatforms := []string{"linux/amd64", "darwin/amd64", "windows/amd64"}

	platform := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	if !sliceContains(supportedPlatforms, platform) {
		t.Skipf("not supported for %s/%s. Supported patforms: %v",
			runtime.GOOS, runtime.GOARCH, supportedPlatforms)
	}

	threads := path.Join("testdata", "threads")

	switch runtime.GOOS {
	case "linux":
		// nothing to do
	case "darwin":
		threads += "-darwin"
	case "windows":
		threads += ".exe"
	}

	var b bytes.Buffer
	cmd := exec.Command(threads)
	cmd.Stdout = &b
	cmd.Stderr = &b

	err := cmd.Start()
	require.NoErrorf(t, err, "failed to start %q", threads)

	var log string
	require.Eventually(t,
		func() bool {
			if cmd.ProcessState != nil {
				t.Fatalf("Process exited with error: '%s'", cmd.ProcessState.String())
			}
			line := b.String()
			log += line
			return strings.Contains(log, "running")
		},
		time.Second, 50*time.Millisecond,
		"could not determine if %q is running. Output: %q",
		threads, log)

	return cmd
}

func initTestResolver() (Stats, error) {
	err := logp.DevelopmentSetup()
	if err != nil {
		return Stats{}, err
	}
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
	err = testConfig.Init()
	return testConfig, err
}

func sliceContains(s []string, e string) bool { //nolint: deadcode,structcheck,unused // needed by other platforms
	for _, v := range s {
		if e == v {
			return true
		}
	}

	return false
}
