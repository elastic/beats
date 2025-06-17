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
	"os"
	"os/user"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-system-metrics/dev-tools/systemtests"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/cgroup"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

func TestFetchOtherProcessCgroup(t *testing.T) {
	_ = logp.DevelopmentSetup()

	testConfig := Stats{
		Procs:        []string{".*"},
		Hostfs:       systemtests.DockerTestResolver(),
		CPUTicks:     false,
		CacheCmdLine: true,
		EnvWhitelist: []string{".*"},
		IncludeTop: IncludeTopConfig{
			Enabled:  false,
			ByCPU:    4,
			ByMemory: 0,
		},
		EnableCgroups: false,
		CgroupOpts: cgroup.ReaderOptions{
			RootfsMountpoint:  systemtests.DockerTestResolver(),
			IgnoreRootCgroups: true,
		},
	}
	err := testConfig.Init()
	assert.NoError(t, err, "Init")

	evts, _, err := testConfig.Get()
	if err != nil {
		assert.ErrorIs(t, err, NonFatalErr{})
	}
	t.Logf("Got %d events", len(evts))
}

func TestGetSelfPidNoHostfs(t *testing.T) {
	ourPid := os.Getpid()

	foundPid, err := GetSelfPid(resolve.NewTestResolver(""))
	require.NoError(t, err)

	require.Equal(t, ourPid, foundPid)

}

func TestFetchProcessFromOtherUser(t *testing.T) {
	_ = logp.DevelopmentSetup()
	// If we just used Get() or FetchPids() to get a list of processes on the system, this would produce a bootstrapping problem
	// where if the code wasn't working (and we were skipping over PIDs not owned by us) this test would pass.
	// re-implement part of the core pid-fetch logic
	// All this does is find a pid that's not owned by us.
	rootpath := systemtests.DockerTestResolver()
	dir, err := os.Open(rootpath.ResolveHostFS("/proc"))
	require.NoError(t, err, "error opening /proc")

	const readAllDirnames = -1 // see os.File.Readdirnames doc

	names, err := dir.Readdirnames(readAllDirnames)
	require.NoError(t, err, "error reading directory names")
	require.NoError(t, dir.Close(), "error closing /proc")
	us, err := user.Current()
	require.NoError(t, err, "error fetching current user")
	var testPid int
	for _, name := range names {
		if name[0] < '0' || name[0] > '9' {
			continue
		}
		pid, err := strconv.Atoi(name)
		if err != nil {
			t.Logf("Error converting PID name %s", name)
			continue
		}
		pidUser, err := getUser(systemtests.DockerTestResolver(), pid)
		if err == nil {
			if pidUser != us.Name {
				testPid = pid
				break
			}
		}
	}
	// CI environments can be weird, we might only have one user
	if testPid == 0 { // can't find any pids that don't belong to us, skip
		t.Logf("found no PIDs from other user, skipping")
		t.SkipNow()
	}

	defer dir.Close()

	testConfig := Stats{
		Procs:        []string{".*"},
		Hostfs:       systemtests.DockerTestResolver(),
		CPUTicks:     false,
		CacheCmdLine: true,
		EnvWhitelist: []string{".*"},
		IncludeTop: IncludeTopConfig{
			Enabled:  false,
			ByCPU:    4,
			ByMemory: 0,
		},
		EnableCgroups: false,
		CgroupOpts: cgroup.ReaderOptions{
			RootfsMountpoint:  systemtests.DockerTestResolver(),
			IgnoreRootCgroups: true,
		},
	}
	err = testConfig.Init()
	assert.NoError(t, err, "Init")

	t.Logf("found process from another user with pid %d, testing", testPid)
	pidData, err := testConfig.GetOne(testPid)
	require.NoError(t, err)
	t.Logf("got: %s", pidData.StringToPrint())
}

func TestGetInfoForPid_NumThreads(t *testing.T) {
	want := ProcState{
		Name:       "elastic-agent",
		State:      "sleeping",
		Pid:        opt.IntWith(42),
		Ppid:       opt.IntWith(1),
		Pgid:       opt.IntWith(4067478),
		NumThreads: opt.IntWith(26),
	}

	got, err := GetInfoForPid(resolve.NewTestResolver("testdata"), 42)
	require.NoError(t, err, "GetInfoForPid returned an error when it should have succeeded")

	assert.Equal(t, want, got)
}

func TestParseProcStat(t *testing.T) {
	data := []byte("4067478 (elastic-agent) S 1 4067478 4067478 0 -1 4194560 151900 " +
		"1587 0 0 8229 3989 0 1 32 12 26" +
		" 0 200791940 2675654656 15487 18446744073709551615 1 1 0 0 0 0 0 0 2143420159 0 0 0 17 9 0 0 0 0 0 0 0 0 0 0 0 0 0")

	want := ProcState{
		Name:       "elastic-agent",
		State:      getProcState(byte('S')),
		Ppid:       opt.IntWith(1),
		Pgid:       opt.IntWith(4067478),
		NumThreads: opt.IntWith(26),
	}

	got, err := parseProcStat(data)
	require.NoError(t, err, "parseProcStat returned and error")

	assert.Equal(t, want, got, "")
}

func TestCgroupsBadCgroupsConfig(t *testing.T) {
	rootfs := systemtests.DockerTestResolver()
	_ = logp.DevelopmentSetup(logp.ToObserverOutput())
	testStats := Stats{CPUTicks: true,
		EnableCgroups: true,
		EnableNetwork: true,
		Hostfs:        rootfs,
		Procs:         []string{".*"},
		CgroupOpts:    cgroup.ReaderOptions{RootfsMountpoint: resolve.NewTestResolver("testdata")}, // procs here have no cgroup data, leading to errors
	}
	err := testStats.Init()
	require.NoError(t, err)

	// make sure we still have proc data despite cgroups errors
	procs, _, err := testStats.Get()
	if err != nil {
		assert.ErrorIs(t, err, NonFatalErr{})
	}

	t.Logf("got %d procs", len(procs))
	require.NotEmpty(t, procs)

	gotLogs := logp.ObserverLogs().TakeAll()
	// check to see if we got the "correct" error message
	foundLogEntry := false

	message := "metrics are valid but partial: error finding cgroup version"
	for _, entry := range gotLogs {
		if strings.Contains(entry.Message, message) {
			foundLogEntry = true
		}
	}
	require.True(t, foundLogEntry, "log line '%s' was not found", message)
}

func TestParseIO(t *testing.T) {
	path := resolve.NewTestResolver("testdata/")
	data, err := getIOData(path, 42)
	require.NoError(t, err)

	good := ProcIOInfo{
		ReadChar:            opt.UintWith(10418),
		WriteChar:           opt.UintWith(8),
		ReadSyscalls:        opt.UintWith(14),
		WriteSyscalls:       opt.UintWith(1),
		ReadBytes:           opt.UintWith(5243),
		WriteBytes:          opt.UintWith(128),
		CancelledWriteBytes: opt.UintWith(4),
	}

	require.Equal(t, good, data)
}
