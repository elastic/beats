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

package cgroup

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/cgroup/testhelpers"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

var testFileList = []string{
	"testdata/docker.zip",
	"testdata/ubuntu1804.zip",
	"testdata/amzn2.zip",
	"testdata/docker2.zip",
}

func TestMain(m *testing.M) {
	os.Exit(testhelpers.MainTestWrapper(m, testFileList))
}

func TestFindMatchingPid(t *testing.T) {
	testFile := `
12
13
14
1585724
1585725
1586244
1586245
`
	got := foundMatchingPidInProcsFile(14, testFile)
	assert.True(t, got)

	gotFalse := foundMatchingPidInProcsFile(15, testFile)

	assert.False(t, gotFalse)
}

func TestFindCgroup(t *testing.T) {
	path, err := guessContainerCgroupPath("/sys/fs/cgroup", os.Getpid())
	require.NoError(t, err)
	t.Logf("got path: %s", path)
}

func TestFindCgroupCache(t *testing.T) {
	testPid := 2233801
	path, err := guessContainerCgroupPath("testdata/docker2/sys/fs/cgroup", testPid)
	goodPath := "/user.slice/user-1000.slice/session-520.scope"
	require.NoError(t, err)
	t.Logf("got path: %s", path)
	require.Equal(t, goodPath, path)

	cached := cgroupContainerPath.get()
	t.Logf("got cached path: %s", cached)
	require.Equal(t, goodPath, cached)

	// run again with cached path
	path, err = guessContainerCgroupPath("testdata/docker2/sys/fs/cgroup", testPid)
	require.NoError(t, err)
	require.Equal(t, goodPath, path)

	// set outdated cache path
	cgroupContainerPath.set("/user.slice/user-1000.slice/session-521.scope")

	// should still get a good path
	path, err = guessContainerCgroupPath("testdata/docker2/sys/fs/cgroup", testPid)
	require.NoError(t, err)
	require.Equal(t, goodPath, path)
	cached = cgroupContainerPath.get()
	require.Equal(t, goodPath, cached)
}

func TestSupportedSubsystems(t *testing.T) {
	subsystems, err := SupportedSubsystems(resolve.NewTestResolver("testdata/docker"))
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, subsystems, 11)
	assertContains(t, subsystems, "cpuset")
	assertContains(t, subsystems, "cpu")
	assertContains(t, subsystems, "cpuacct")
	assertContains(t, subsystems, "blkio")
	assertContains(t, subsystems, "memory")
	assertContains(t, subsystems, "devices")
	assertContains(t, subsystems, "freezer")
	assertContains(t, subsystems, "net_cls")
	assertContains(t, subsystems, "perf_event")
	assertContains(t, subsystems, "net_prio")
	assertContains(t, subsystems, "pids")

	_, found := subsystems["hugetlb"]
	assert.False(t, found, "hugetlb should be missing because it's disabled")
}

func TestSupportedSubsystemsErrCgroupsMissing(t *testing.T) {
	_, err := SupportedSubsystems(resolve.NewTestResolver("testdata/doesnotexist"))
	if !errors.Is(err, ErrCgroupsMissing) {
		t.Fatalf("expected ErrCgroupsMissing, but got %v", err)
	}
}

func TestSubsystemMountpoints(t *testing.T) {
	subsystems := map[string]struct{}{}
	subsystems["blkio"] = struct{}{}
	subsystems["cpu"] = struct{}{}
	subsystems["cpuacct"] = struct{}{}
	subsystems["cpuset"] = struct{}{}
	subsystems["devices"] = struct{}{}
	subsystems["freezer"] = struct{}{}
	subsystems["hugetlb"] = struct{}{}
	subsystems["memory"] = struct{}{}
	subsystems["perf_event"] = struct{}{}

	mountpoints, err := SubsystemMountpoints(resolve.NewTestResolver("testdata/docker"), subsystems, logptest.NewTestingLogger(t, ""))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "testdata/docker/sys/fs/cgroup/blkio", mountpoints.V1Mounts["blkio"])
	assert.Equal(t, "testdata/docker/sys/fs/cgroup/cpu", mountpoints.V1Mounts["cpu"])
	assert.Equal(t, "testdata/docker/sys/fs/cgroup/cpuacct", mountpoints.V1Mounts["cpuacct"])
	assert.Equal(t, "testdata/docker/sys/fs/cgroup/cpuset", mountpoints.V1Mounts["cpuset"])
	assert.Equal(t, "testdata/docker/sys/fs/cgroup/devices", mountpoints.V1Mounts["devices"])
	assert.Equal(t, "testdata/docker/sys/fs/cgroup/freezer", mountpoints.V1Mounts["freezer"])
	assert.Equal(t, "testdata/docker/sys/fs/cgroup/hugetlb", mountpoints.V1Mounts["hugetlb"])
	assert.Equal(t, "testdata/docker/sys/fs/cgroup/memory", mountpoints.V1Mounts["memory"])
	assert.Equal(t, "testdata/docker/sys/fs/cgroup/perf_event", mountpoints.V1Mounts["perf_event"])
}

func TestProcessCgroupPaths(t *testing.T) {
	reader, err := NewReaderOptions(ReaderOptions{
		RootfsMountpoint: resolve.NewTestResolver("testdata/docker"),
		Logger:           logptest.NewTestingLogger(t, ""),
	})
	if err != nil {
		t.Fatalf("error in NewReader: %s", err)
	}
	paths, err := reader.ProcessCgroupPaths(985)
	if err != nil {
		t.Fatalf("error in ProcessCgroupPaths: %s", err)
	}

	path := "/docker/b29faf21b7eff959f64b4192c34d5d67a707fe8561e9eaa608cb27693fba4242"
	assert.Equal(t, path, paths.V1["blkio"].ControllerPath)
	assert.Equal(t, path, paths.V1["cpu"].ControllerPath)
	assert.Equal(t, path, paths.V1["cpuacct"].ControllerPath)
	assert.Equal(t, path, paths.V1["cpuset"].ControllerPath)
	assert.Equal(t, path, paths.V1["devices"].ControllerPath)
	assert.Equal(t, path, paths.V1["freezer"].ControllerPath)
	assert.Equal(t, path, paths.V1["memory"].ControllerPath)
	assert.Equal(t, path, paths.V1["net_cls"].ControllerPath)
	assert.Equal(t, path, paths.V1["net_prio"].ControllerPath)
	assert.Equal(t, path, paths.V1["perf_event"].ControllerPath)
	assert.Len(t, paths.Flatten(), 10)
}

func TestProcessCgroupHybridPaths(t *testing.T) {
	reader, err := NewReaderOptions(ReaderOptions{
		RootfsMountpoint: resolve.NewTestResolver("testdata/amzn2"),
		Logger:           logptest.NewTestingLogger(t, ""),
	})
	if err != nil {
		t.Fatalf("error in NewReader: %s", err)
	}
	paths, err := reader.ProcessCgroupPaths(493239)
	if err != nil {
		t.Fatalf("error in ProcessCgroupPaths: %s", err)
	}

	path := "/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod7a96c459_d529_44ae_9f99_90d3798d6426.slice/cri-containerd-1d3d308a7d48a27814a68bf33a44acf4441c9c02463ca0bc1cdfdc8c0b4a8496.scope"
	assert.Equal(t, path, paths.V1["blkio"].ControllerPath)
	assert.Equal(t, path, paths.V1["cpu"].ControllerPath)
	assert.Equal(t, path, paths.V1["cpuacct"].ControllerPath)
	assert.Equal(t, path, paths.V1["cpuset"].ControllerPath)
	assert.Equal(t, path, paths.V1["devices"].ControllerPath)
	assert.Equal(t, path, paths.V1["freezer"].ControllerPath)
	assert.Equal(t, path, paths.V1["memory"].ControllerPath)
	assert.Equal(t, path, paths.V1["net_cls"].ControllerPath)
	assert.Equal(t, path, paths.V1["net_prio"].ControllerPath)
	assert.Equal(t, path, paths.V1["perf_event"].ControllerPath)
	assert.Equal(t, path, paths.V1["hugetlb"].ControllerPath)
	assert.Len(t, paths.Flatten(), 13)
}

func TestProcessCgroupPathsV2(t *testing.T) {
	reader, err := NewReaderOptions(ReaderOptions{
		RootfsMountpoint: resolve.NewTestResolver("testdata/docker"),
		Logger:           logptest.NewTestingLogger(t, ""),
	})
	if err != nil {
		t.Fatalf("error in NewReader: %s", err)
	}

	paths, err := reader.ProcessCgroupPaths(312)
	if err != nil {
		t.Fatalf("error in ProcessCgroupPaths: %s", err)
	}

	assert.Equal(t, "testdata/docker/sys/fs/cgroup/system.slice/docker-1c8fa019edd4b9d4b2856f4932c55929c5c118c808ed5faee9a135ca6e84b039.scope", paths.V2["cgroup"].FullPath)
	assert.Equal(t, "testdata/docker/sys/fs/cgroup/system.slice/docker-1c8fa019edd4b9d4b2856f4932c55929c5c118c808ed5faee9a135ca6e84b039.scope", paths.V2["cpu"].FullPath)
	assert.Equal(t, "testdata/docker/sys/fs/cgroup/system.slice/docker-1c8fa019edd4b9d4b2856f4932c55929c5c118c808ed5faee9a135ca6e84b039.scope", paths.V2["io"].FullPath)
	assert.Equal(t, "testdata/docker/sys/fs/cgroup/system.slice/docker-1c8fa019edd4b9d4b2856f4932c55929c5c118c808ed5faee9a135ca6e84b039.scope", paths.V2["memory"].FullPath)
}

func TestMountpointsV2(t *testing.T) {
	// emulate running in a private namespace docker container
	cgroupNSStateFetch = func(*logp.Logger) bool { return true }
	// inject our PID into the cgroup.procs file to so ProcessCgroupPaths()
	// can find our root cgroup
	pid := os.Getpid()
	pidFmt := fmt.Sprintf("%d\n", pid)
	err := os.WriteFile("testdata/docker2/sys/fs/cgroup/user.slice/user-1000.slice/session-520.scope/cgroup.procs",
		[]byte(pidFmt), 0o744)
	require.NoError(t, err)

	reader, err := NewReaderOptions(ReaderOptions{
		RootfsMountpoint: resolve.NewTestResolver("testdata/docker2"),
		Logger:           logptest.NewTestingLogger(t, ""),
	})
	require.NoError(t, err)

	stats, err := reader.GetStatsForPid(2233801)
	require.NoError(t, err)
	// unpack the interface so we can test this a little better
	rawObject, ok := stats.(*StatsV2)
	require.True(t, ok)
	require.Equal(t, rawObject.ID, "session-520.scope")
	require.Equal(t, rawObject.Path, "/user.slice/user-1000.slice/session-520.scope")
}

func assertContains(t testing.TB, m map[string]struct{}, key string) {
	_, contains := m[key]
	if !contains {
		t.Errorf("map is missing key %v, map=%+v", key, m)
	}
}

func TestParseMountinfoLine(t *testing.T) {
	lines := []string{
		"30 24 0:25 / /sys/fs/cgroup/blkio rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,blkio",
		"30 24 0:25 / /sys/fs/cgroup/blkio rw,nosuid,nodev,noexec,relatime shared:13 - cgroup cgroup rw,blkio",
		"30 24 0:25 / /sys/fs/cgroup/blkio rw,nosuid,nodev,noexec,relatime shared:13 master:1 - cgroup cgroup rw,blkio",
		"30 24 0:25 / /sys/fs/cgroup/blkio rw,nosuid,nodev,noexec,relatime shared:13 - cgroup cgroup rw,name=blkio",
	}

	for _, line := range lines {
		mount, err := parseMountinfoLine(line)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "/sys/fs/cgroup/blkio", mount.mountpoint)
		assert.Equal(t, "cgroup", mount.filesystemType)
		assert.Len(t, mount.superOptions, 2)
	}
}

func TestFetchV2Paths(t *testing.T) {
	cases := []struct {
		name         string
		lines        []string
		rootfs       resolve.Resolver
		expectedPath string
	}{
		{
			name:   "hostfspaths-with-hostfs",
			rootfs: resolve.NewTestResolver("/hostfs"),
			lines: []string{
				"/sys/fs/cgroup",
				"/hostfs/sys/fs/cgroup",
				"/hostfs/var/lib/docker/overlay2/1b570230fa3ec3679e354b0c219757c739f91d774ebc02174106488606549da0/merged/sys/fs/cgroup",
			},
			expectedPath: "/hostfs/sys/fs/cgroup",
		},
		{
			name:   "hostfspaths-without-hostfs",
			rootfs: resolve.NewTestResolver(""),
			lines: []string{
				"/sys/fs/cgroup",
				"/hostfs/sys/fs/cgroup",
				"/hostfs/var/lib/docker/overlay2/1b570230fa3ec3679e354b0c219757c739f91d774ebc02174106488606549da0/merged/sys/fs/cgroup",
			},
			expectedPath: "/hostfs/sys/fs/cgroup",
		},
		{
			name:   "hostfspaths-with-hostfs-werid-order",
			rootfs: resolve.NewTestResolver("/hostfs"),
			lines: []string{
				"/sys/fs/cgroup",
				"/hostfs/var/lib/docker/overlay2/1b570230fa3ec3679e354b0c219757c739f91d774ebc02174106488606549da0/merged/sys/fs/cgroup",
				"/hostfs/sys/fs/cgroup",
			},
			expectedPath: "/hostfs/sys/fs/cgroup",
		},
		{
			name:   "hostfspaths-with-hostfs-werider-order",
			rootfs: resolve.NewTestResolver("/hostfs"),
			lines: []string{
				"/sys/fs/cgroup",
				"/hostfs/sys/fs/cgroup",
				"/hostfs/var/lib/docker/overlay2/1b570230fa3ec3679e354b0c219757c739f91d774ebc02174106488606549da0/merged/sys/fs/cgroup",
			},
			expectedPath: "/hostfs/sys/fs/cgroup",
		},
		{
			name:         "no-hostfs-normalv2",
			rootfs:       resolve.NewTestResolver(""),
			lines:        []string{"/sys/fs/cgroup"},
			expectedPath: "/sys/fs/cgroup",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := getProperV2Paths(testCase.rootfs, testCase.lines, logptest.NewTestingLogger(t, ""))
			assert.Equal(t, testCase.expectedPath, got)
		})
	}
}
