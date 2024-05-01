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

package cgv2

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

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
}

func TestGetCPU(t *testing.T) {
	cpu := CPUSubsystem{}
	err := cpu.Get(v2Path)
	assert.NoError(t, err, "error in Get")

	assert.Equal(t, uint64(26772130245), cpu.Stats.Usage.NS)
	assert.Equal(t, uint64(5793060316), cpu.Stats.System.NS)
}
