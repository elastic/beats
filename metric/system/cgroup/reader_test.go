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
// +build linux

package cgroup

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

const (
	path = "/docker/b29faf21b7eff959f64b4192c34d5d67a707fe8561e9eaa608cb27693fba4242"
	id   = "b29faf21b7eff959f64b4192c34d5d67a707fe8561e9eaa608cb27693fba4242"

	pathv2 = "/system.slice/docker-1c8fa019edd4b9d4b2856f4932c55929c5c118c808ed5faee9a135ca6e84b039.scope"
	idv2   = "docker-1c8fa019edd4b9d4b2856f4932c55929c5c118c808ed5faee9a135ca6e84b039.scope"
)

func TestV1EventDifferentPaths(t *testing.T) {
	pid := 3757
	reader, err := NewReader(resolve.NewTestResolver("testdata/ubuntu1804"), true)
	require.NoError(t, err, "error in NewReader")

	stats, err := reader.GetV1StatsForProcess(pid)
	require.NoError(t, err, "error in GetV1StatsForProcess")

	require.NotNil(t, stats, "no cgroup stats found")

	// Make sure we can handle root paths properly
	require.Equal(t, "/system.slice/networkd-dispatcher.service", stats.Path)
	require.Equal(t, "networkd-dispatcher.service", stats.ID)
}

func TestReaderGetStatsV1(t *testing.T) {
	reader, err := NewReader(resolve.NewTestResolver("testdata/docker"), true)
	require.NoError(t, err, "error in NewReader")

	stats, err := reader.GetV1StatsForProcess(985)
	require.NoError(t, err, "error in GetV1StatsForProcess")

	require.NotNil(t, stats, "no cgroup stats found")

	require.Equal(t, id, stats.ID)
	require.Equal(t, id, stats.BlockIO.ID)
	require.Equal(t, id, stats.CPU.ID)
	require.Equal(t, id, stats.CPUAccounting.ID)
	require.Equal(t, id, stats.Memory.ID)

	require.NotZero(t, stats.CPU.CFS.PeriodMicros.Us)
	require.NotZero(t, stats.CPUAccounting.Total.NS)
	require.NotZero(t, stats.Memory.Mem.Usage.Bytes)
	require.NotZero(t, stats.BlockIO.Total.Bytes)

	require.Equal(t, path, stats.Path)
	require.Equal(t, path, stats.BlockIO.Path)
	require.Equal(t, path, stats.CPU.Path)
	require.Equal(t, path, stats.CPUAccounting.Path)
	require.Equal(t, path, stats.Memory.Path)

}

func TestReaderGetStatsV2(t *testing.T) {
	reader, err := NewReader(resolve.NewTestResolver("testdata/docker"), true)
	require.NoError(t, err, "error in NewReader")

	stats, err := reader.GetV2StatsForProcess(312)
	require.NoError(t, err, "error in GetV2StatsForProcess")

	require.NotNil(t, stats.CPU)
	require.NotNil(t, stats.Memory)
	require.NotNil(t, stats.IO)

	require.Equal(t, pathv2, stats.Path)
	require.Equal(t, idv2, stats.ID)

	require.NotZero(t, stats.CPU.Stats.Usage.NS)
	require.NotZero(t, stats.Memory.Mem.Usage.Bytes)
	require.NotZero(t, stats.IO.Pressure["some"].Sixty.Pct)

}

func TestReaderGetStatsHierarchyOverride(t *testing.T) {
	// In testdata/docker, process 1's cgroup paths have
	// no corresponding paths under /sys/fs/cgroup/<subsystem>.
	//
	// Setting CgroupsHierarchyOverride means that we use
	// the root cgroup path instead. This is intended to test
	// the scenario where we're reading cgroup metrics from
	// within a Docker container.

	reader, err := NewReaderOptions(ReaderOptions{
		RootfsMountpoint:         resolve.NewTestResolver("testdata/docker"),
		IgnoreRootCgroups:        false,
		CgroupsHierarchyOverride: "/",
	})
	require.NoError(t, err, "error in NewReaderOptions")

	stats, err := reader.GetV1StatsForProcess(1)
	require.NoError(t, err, "error in GetV1StatsForProcess")

	require.NotNil(t, stats, "no cgroup stats found")

	require.NotNil(t, stats.CPU, "no cpu metrics")
	require.NotZero(t, stats.CPU.CFS.Shares, "no V1 CFS cpu metrics")

	reader2, err := NewReaderOptions(ReaderOptions{
		RootfsMountpoint:         resolve.NewTestResolver("testdata/docker"),
		IgnoreRootCgroups:        true,
		CgroupsHierarchyOverride: "/system.slice/",
	})
	require.NoError(t, err, "error in NewReaderOptions")

	stats2, err := reader2.GetV2StatsForProcess(312)
	require.NoError(t, err, "error in GetV2StatsForProcess")

	require.NotNil(t, stats, "no cgroup stats found")

	require.NotNil(t, stats2.CPU, "no v2 cpu stats found")
	require.NotZero(t, stats2.CPU.Stats.Usage.NS, "no v2 CPU usage stats")
}
