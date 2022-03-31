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

package cgroup

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
)

const (
	path = "/docker/b29faf21b7eff959f64b4192c34d5d67a707fe8561e9eaa608cb27693fba4242"
	id   = "b29faf21b7eff959f64b4192c34d5d67a707fe8561e9eaa608cb27693fba4242"
)

func TestReaderGetStatsV1(t *testing.T) {
	reader, err := NewReader(resolve.NewTestResolver("testdata/docker"), true)
	assert.NoError(t, err, "error in NewReader")

	stats, err := reader.GetV1StatsForProcess(985)
	assert.NoError(t, err, "error in GetV1StatsForProcess")

	if stats == nil {
		t.Fatal("no cgroup stats found")
	}

	assert.Equal(t, id, stats.ID)
	assert.Equal(t, id, stats.BlockIO.ID)
	assert.Equal(t, id, stats.CPU.ID)
	assert.Equal(t, id, stats.CPUAccounting.ID)
	assert.Equal(t, id, stats.Memory.ID)

	assert.NotZero(t, stats.CPU.CFS.PeriodMicros.Us)
	assert.NotZero(t, stats.CPUAccounting.Total.NS)
	assert.NotZero(t, stats.Memory.Mem.Usage.Bytes)
	assert.NotZero(t, stats.BlockIO.Total.Bytes)

	assert.Equal(t, path, stats.Path)
	assert.Equal(t, path, stats.BlockIO.Path)
	assert.Equal(t, path, stats.CPU.Path)
	assert.Equal(t, path, stats.CPUAccounting.Path)
	assert.Equal(t, path, stats.Memory.Path)

	json, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(json))
}

func TestReaderGetStatsV2(t *testing.T) {
	reader, err := NewReader(resolve.NewTestResolver("testdata/docker"), true)
	assert.NoError(t, err, "error in NewReader")

	stats, err := reader.GetV2StatsForProcess(312)
	assert.NoError(t, err, "error in GetV2StatsForProcess")

	require.NotNil(t, stats.CPU)
	require.NotNil(t, stats.Memory)
	require.NotNil(t, stats.IO)

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
	if err != nil {
		t.Fatal(err)
	}

	stats, err := reader.GetV1StatsForProcess(1)
	if err != nil {
		t.Fatal(err)
	}
	if stats == nil {
		t.Fatal("no cgroup stats found")
	}

	require.NotNil(t, stats.CPU)
	assert.NotZero(t, stats.CPU.CFS.Shares)

	reader2, err := NewReaderOptions(ReaderOptions{
		RootfsMountpoint:         resolve.NewTestResolver("testdata/docker"),
		IgnoreRootCgroups:        true,
		CgroupsHierarchyOverride: "/system.slice/",
	})
	if err != nil {
		t.Fatal(err)
	}

	stats2, err := reader2.GetV2StatsForProcess(312)
	if err != nil {
		t.Fatal(err)
	}
	if stats == nil {
		t.Fatal("no cgroup stats found")
	}

	require.NotNil(t, stats2.CPU)
	require.NotZero(t, stats2.CPU.Stats.Usage.NS)
}
