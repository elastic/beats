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

package memory

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-system-metrics/dev-tools/systemtests"
)

// zswapExpectation defines expected zswap behavior for a CI environment
type zswapExpectation struct {
	zswapExists bool // Whether Zswap/Zswapped fields exist in /proc/meminfo
	debugExists bool // Whether /sys/kernel/debug/zswap is accessible
}

// ciExpectations maps BUILDKITE_STEP_KEY to expected zswap behavior.
// Keys must match the `key` field in .buildkite/pipeline.yml
var ciExpectations = map[string]zswapExpectation{
	"linux-container-test":       {zswapExists: true, debugExists: false},  // Ubuntu 22.04: modern kernel, zswap in meminfo, no debugfs
	"linux-container-test-rhel9": {zswapExists: true, debugExists: false},  // RHEL 9: modern kernel, zswap in meminfo, no debugfs
	"linux-container-test-u2004": {zswapExists: false, debugExists: true},  // Ubuntu 20.04: older kernel, no meminfo but debugfs accessible
	"linux-test":                 {zswapExists: false, debugExists: false}, // Unit tests, unprivileged
	// Test locally with:
	// go test -c ./metric/memory -o memory.test
	// sudo BUILDKITE_STEP_KEY=manual PRIVILEGED=1 ./memory.test -test.run TestMemoryFromContainer
	"manual": {zswapExists: true, debugExists: true},
}

// TestMemoryFromContainer tests memory metric collection from inside a container
// monitoring the host via /hostfs mount
func TestMemoryFromContainer(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	hostfs := systemtests.DockerTestResolver(logger)

	mem, err := Get(hostfs)
	require.NoError(t, err)

	// Basic memory metrics should always be available
	assert.True(t, mem.Total.Exists(), "Total memory should exist")
	assert.True(t, mem.Free.Exists(), "Free memory should exist")
	assert.True(t, mem.Used.Bytes.Exists(), "Used memory should exist")
	assert.True(t, mem.Actual.Free.Exists(), "Actual free memory should exist")

	t.Logf("Total: %d, Free: %d, Used: %d", mem.Total.ValueOr(0), mem.Free.ValueOr(0), mem.Used.Bytes.ValueOr(0))

	zswapExists := mem.Zswap.Compressed.Exists()
	debugExists := !mem.Zswap.Debug.IsZero()

	stepKey := os.Getenv("BUILDKITE_STEP_KEY")
	t.Logf("Zswap exists: %v, Debug exists: %v (BUILDKITE_STEP_KEY=%q)", zswapExists, debugExists, stepKey)

	logZswapStatus(t, mem, zswapExists, debugExists)
	if stepKey == "" {
		// Not in CI or step key not set: fallback to non-enforcing behavior
		return
	}

	expected, ok := ciExpectations[stepKey]
	require.Truef(t, ok, `BUILDKITE_STEP_KEY=%q not found in ciExpectations map.

To fix this test:
1. Check the debug output above for "Zswap exists" and "Debug exists" values
2. Look at the CI print_debug_info() output for kernel config and zswap status
3. Add an entry to ciExpectations in memory_integration_test.go:
   %q: {zswapExists: <true|false>, debugExists: <true|false>}`,
		stepKey, stepKey,
	)

	// Enforce expectations
	if expected.zswapExists {
		assert.True(t, zswapExists, "expected zswap metrics in /proc/meminfo for step %q", stepKey)
		assert.True(t, mem.Zswap.Uncompressed.Exists())
	} else {
		assert.False(t, zswapExists, "expected NO zswap metrics in /proc/meminfo for step %q", stepKey)
	}

	if expected.debugExists {
		assert.True(t, debugExists, "expected debug metrics accessible for step %q", stepKey)
		assert.NotEmpty(t, os.Getenv("PRIVILEGED"), "debugfs access requires PRIVILEGED")
	} else {
		assert.False(t, debugExists, "expected NO debug metrics accessible for step %q", stepKey)
	}
}

func logZswapStatus(t *testing.T, mem Memory, zswapExists, debugExists bool) {
	t.Helper()
	if zswapExists {
		t.Logf("Zswap: Compressed=%d bytes, Uncompressed=%d bytes",
			mem.Zswap.Compressed.ValueOr(0), mem.Zswap.Uncompressed.ValueOr(0))
	} else {
		t.Log("Zswap is not available on this system")
	}

	if debugExists {
		t.Logf("Zswap debug: StoredPages=%d, PoolTotalSize=%d",
			mem.Zswap.Debug.StoredPages.ValueOr(0), mem.Zswap.Debug.PoolTotalSize.ValueOr(0))
	} else {
		t.Log("Zswap debug metrics not accessible")
	}
}
