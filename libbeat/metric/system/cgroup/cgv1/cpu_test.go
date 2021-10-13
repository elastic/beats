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

const cpuPath = "../testdata/docker/sys/fs/cgroup/cpu/docker/b29faf21b7eff959f64b4192c34d5d67a707fe8561e9eaa608cb27693fba4242"

func TestCpuStats(t *testing.T) {
	cpu := CPUSubsystem{}
	if err := cpuStat(cpuPath, &cpu); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, uint64(769021), cpu.Stats.Periods)
	assert.Equal(t, uint64(1046), cpu.Stats.Throttled.Periods)
	assert.Equal(t, uint64(352597023453), cpu.Stats.Throttled.Us)
}

func TestCpuCFS(t *testing.T) {
	cpu := CPUSubsystem{}
	if err := cpuCFS(cpuPath, &cpu); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, uint64(100000), cpu.CFS.PeriodMicros.Us)
	assert.Equal(t, uint64(0), cpu.CFS.QuotaMicros.Us) // -1 is changed to 0.
	assert.Equal(t, uint64(1024), cpu.CFS.Shares)
}

func TestCpuRT(t *testing.T) {
	cpu := CPUSubsystem{}
	if err := cpuRT(cpuPath, &cpu); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, uint64(1000000), cpu.RT.Period.Us)
	assert.Equal(t, uint64(0), cpu.RT.Runtime.Us)
}

func TestCpuSubsystemGet(t *testing.T) {
	cpu := CPUSubsystem{}
	if err := cpu.Get(cpuPath); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, uint64(769021), cpu.Stats.Periods)
	assert.Equal(t, uint64(100000), cpu.CFS.PeriodMicros.Us)
	assert.Equal(t, uint64(1000000), cpu.RT.Period.Us)
}

func TestCpuSubsystemJSON(t *testing.T) {
	cpu := CPUSubsystem{}
	if err := cpu.Get(cpuPath); err != nil {
		t.Fatal(err)
	}

	json, err := json.MarshalIndent(cpu, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(json))
}
