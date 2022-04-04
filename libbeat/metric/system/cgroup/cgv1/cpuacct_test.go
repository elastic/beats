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

const cpuacctPath = "../testdata/docker/sys/fs/cgroup/cpuacct/docker/b29faf21b7eff959f64b4192c34d5d67a707fe8561e9eaa608cb27693fba4242"

func TestCPUAccountingStats(t *testing.T) {
	cpuacct := CPUAccountingSubsystem{}
	if err := cpuacctStat(cpuacctPath, &cpuacct); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, uint64(61950000000), cpuacct.Stats.User.NS)
	assert.Equal(t, uint64(7730000000), cpuacct.Stats.System.NS)
}

func TestCpuacctUsage(t *testing.T) {
	cpuacct := CPUAccountingSubsystem{}
	if err := cpuacctUsage(cpuacctPath, &cpuacct); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, uint64(95996653175), cpuacct.Total.NS)
}

func TestCpuacctUsagePerCPU(t *testing.T) {
	cpuacct := CPUAccountingSubsystem{}
	if err := cpuacctUsagePerCPU(cpuacctPath, &cpuacct); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, map[string]uint64{"1": 0x62fcde13c, "2": 0x565f2fcaa, "3": 0x5a8736ea1, "4": 0x51b92ac82}, cpuacct.UsagePerCPU)
}

func TestCPUAccountingSubsystem_Get(t *testing.T) {
	cpuacct := CPUAccountingSubsystem{}
	if err := cpuacct.Get(cpuacctPath); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, uint64(61950000000), cpuacct.Stats.User.NS)
	assert.Equal(t, uint64(95996653175), cpuacct.Total.NS)
	assert.Len(t, cpuacct.UsagePerCPU, 4)
}

func TestCPUAccountingSubsystemJSON(t *testing.T) {
	cpuacct := CPUAccountingSubsystem{}
	if err := cpuacct.Get(cpuacctPath); err != nil {
		t.Fatal(err)
	}

	json, err := json.MarshalIndent(cpuacct, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(json))
}
