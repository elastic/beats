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

package cgv1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

const memoryPath = "../testdata/docker/sys/fs/cgroup/memory/docker/b29faf21b7eff959f64b4192c34d5d67a707fe8561e9eaa608cb27693fba4242"

func TestMemoryStat(t *testing.T) {
	mem := MemorySubsystem{}
	if err := memoryStats(memoryPath, &mem); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, uint64(65101824), mem.Stats.Cache.Bytes)
	assert.Equal(t, uint64(230662144), mem.Stats.RSS.Bytes)
	assert.Equal(t, uint64(174063616), mem.Stats.RSSHuge.Bytes)
	assert.Equal(t, uint64(17633280), mem.Stats.MappedFile.Bytes)
	assert.Equal(t, uint64(0), mem.Stats.Swap.Bytes)
	assert.Equal(t, uint64(103258), mem.Stats.PagesIn)
	assert.Equal(t, uint64(77551), mem.Stats.PagesOut)
	assert.Equal(t, uint64(91651), mem.Stats.PageFaults)
	assert.Equal(t, uint64(166), mem.Stats.MajorPageFaults)
	assert.Equal(t, uint64(28672), mem.Stats.InactiveAnon.Bytes)
	assert.Equal(t, uint64(230780928), mem.Stats.ActiveAnon.Bytes)
	assert.Equal(t, uint64(40108032), mem.Stats.InactiveFile.Bytes)
	assert.Equal(t, uint64(24813568), mem.Stats.ActiveFile.Bytes)
	assert.Equal(t, uint64(0), mem.Stats.Unevictable.Bytes)
	assert.Equal(t, uint64(9223372036854771712), mem.Stats.HierarchicalMemoryLimit.Bytes)
	assert.Equal(t, uint64(9223372036854771712), mem.Stats.HierarchicalMemswLimit.Bytes)
}

func TestMemoryData(t *testing.T) {
	usage := MemoryData{}
	if err := memoryData(memoryPath, "memory", &usage); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, uint64(295997440), usage.Usage.Bytes)
	assert.Equal(t, uint64(298532864), usage.Usage.Max.Bytes)
	assert.Equal(t, uint64(9223372036854771712), usage.Limit.Bytes)
	assert.Equal(t, uint64(0), usage.Failures)
}

func TestMemoryDataSwap(t *testing.T) {
	usage := MemoryData{}
	if err := memoryData(memoryPath, "memory.memsw", &usage); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, uint64(295997440), usage.Usage.Bytes)
	assert.Equal(t, uint64(298532864), usage.Usage.Max.Bytes)
	assert.Equal(t, uint64(9223372036854771712), usage.Limit.Bytes)
	assert.Equal(t, uint64(0), usage.Failures)
}

func TestMemoryDataKernel(t *testing.T) {
	usage := MemoryData{}
	if err := memoryData(memoryPath, "memory.kmem", &usage); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, uint64(40), usage.Usage.Bytes)
	assert.Equal(t, uint64(50), usage.Usage.Max.Bytes)
	assert.Equal(t, uint64(9223372036854771712), usage.Limit.Bytes)
	assert.Equal(t, uint64(0), usage.Failures)
}

func TestMemoryDataKernelTCP(t *testing.T) {
	usage := MemoryData{}
	if err := memoryData(memoryPath, "memory.kmem.tcp", &usage); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, uint64(10), usage.Usage.Bytes)
	assert.Equal(t, uint64(70), usage.Usage.Max.Bytes)
	assert.Equal(t, uint64(9223372036854771712), usage.Limit.Bytes)
	assert.Equal(t, uint64(0), usage.Failures)
}

func TestMemorySubsystemGet(t *testing.T) {
	mem := MemorySubsystem{}
	if err := mem.Get(memoryPath); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, uint64(65101824), mem.Stats.Cache.Bytes)
	assert.Equal(t, uint64(295997440), mem.Mem.Usage.Bytes)
	assert.Equal(t, uint64(295997440), mem.MemSwap.Usage.Bytes)
	assert.Equal(t, uint64(40), mem.Kernel.Usage.Bytes)
	assert.Equal(t, uint64(10), mem.KernelTCP.Usage.Bytes)
}

func TestMemorySubsystemJSON(t *testing.T) {
	mem := MemorySubsystem{}
	if err := mem.Get(memoryPath); err != nil {
		t.Fatal(err)
	}

	json, err := json.MarshalIndent(mem, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(json))
}
