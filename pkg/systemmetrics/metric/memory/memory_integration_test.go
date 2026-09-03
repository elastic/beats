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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/systemmetrics/dev-tools/systemtests"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// zswapExpectation defines expected zswap behavior for a CI environment
type zswapExpectation struct {
	zswapExists bool // Whether Zswap/Zswapped fields exist in /proc/meminfo
	debugExists bool // Whether /sys/kernel/debug/zswap is accessible
}

// ciExpectations maps a CI step to expected zswap behavior.
//
// Keys are either a bare BUILDKITE_STEP_KEY, or "<BUILDKITE_STEP_KEY>/<image family>"
// for matrix steps where the same step key runs on several host images. Step keys
// must match the `key` field and image names the `matrix` values in
// .buildkite/pkg/systemmetrics/pipeline.yml.
var ciExpectations = map[string]zswapExpectation{
	// Ubuntu 22.04 host: kernel >= 5.19 reports Zswap in /proc/meminfo, debugfs not readable
	"mandatory-systemmetrics-unit-test":                   {zswapExists: true, debugExists: false},
	"mandatory-systemmetrics-container-tests/ubuntu-2204": {zswapExists: true, debugExists: false},
	// RHEL 9 host: zswap backported into /proc/meminfo, debugfs not readable
	"mandatory-systemmetrics-container-tests/rhel-9": {zswapExists: true, debugExists: false},
	// Ubuntu 20.04 host: kernel < 5.19 has no Zswap fields in /proc/meminfo,
	// but /sys/kernel/debug/zswap is readable from a privileged container
	"mandatory-systemmetrics-container-tests/ubuntu-2004": {zswapExists: false, debugExists: true},
	// Test locally with:
	// go test -c ./pkg/systemmetrics/metric/memory -o memory.test
	// sudo BUILDKITE_STEP_KEY=manual PRIVILEGED=1 ./memory.test -test.run TestMemoryFromContainer
	"manual": {zswapExists: true, debugExists: true},
}

// ciImageFamilies are substrings that identify the host image family in the
// SYSTEMMETRICS_CI_IMAGE value (e.g. "platform-ingest-beats-ubuntu-2004-1782878510").
// The numeric suffix rotates with every image bump, so match on the family only.
var ciImageFamilies = []string{"ubuntu-2004", "ubuntu-2204", "rhel-9"}

// ciExpectationKey builds the ciExpectations lookup key. When the image family is
// known the step-specific "<step>/<family>" key is used, otherwise the bare step key.
func ciExpectationKey(stepKey, image string) string {
	for _, family := range ciImageFamilies {
		if strings.Contains(image, family) {
			return stepKey + "/" + family
		}
	}
	return stepKey
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
	image := os.Getenv("SYSTEMMETRICS_CI_IMAGE")
	t.Logf("Zswap exists: %v, Debug exists: %v (BUILDKITE_STEP_KEY=%q, SYSTEMMETRICS_CI_IMAGE=%q)",
		zswapExists, debugExists, stepKey, image)

	logZswapStatus(t, mem, zswapExists, debugExists)
	if stepKey == "" {
		// Not in CI or step key not set: fallback to non-enforcing behavior
		return
	}

	key := ciExpectationKey(stepKey, image)
	expected, ok := ciExpectations[key]
	require.Truef(t, ok, `BUILDKITE_STEP_KEY=%q, SYSTEMMETRICS_CI_IMAGE=%q (key %q) not found in ciExpectations map.

To fix this test:
1. Check the debug output above for "Zswap exists" and "Debug exists" values
2. If the image is a new family, add it to ciImageFamilies in memory_integration_test.go
3. Add an entry to ciExpectations in memory_integration_test.go:
   %q: {zswapExists: <true|false>, debugExists: <true|false>}
   Step keys and matrix images are defined in .buildkite/pkg/systemmetrics/pipeline.yml`,
		stepKey, image, key, key,
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

func TestCIExpectationKey(t *testing.T) {
	cases := []struct {
		name    string
		stepKey string
		image   string
		want    string
	}{
		{
			name:    "matrix image ubuntu 20.04",
			stepKey: "mandatory-systemmetrics-container-tests",
			image:   "platform-ingest-beats-ubuntu-2004-1782878510",
			want:    "mandatory-systemmetrics-container-tests/ubuntu-2004",
		},
		{
			name:    "matrix image rhel 9",
			stepKey: "mandatory-systemmetrics-container-tests",
			image:   "platform-ingest-beats-rhel-9-1782878510",
			want:    "mandatory-systemmetrics-container-tests/rhel-9",
		},
		{
			name:    "no image falls back to step key",
			stepKey: "mandatory-systemmetrics-unit-test",
			image:   "",
			want:    "mandatory-systemmetrics-unit-test",
		},
		{
			name:    "unknown image family falls back to step key",
			stepKey: "manual",
			image:   "some-other-image",
			want:    "manual",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ciExpectationKey(tc.stepKey, tc.image)
			assert.Equal(t, tc.want, got, "unexpected ciExpectations key")
			if tc.image != "" && tc.want != tc.stepKey {
				_, ok := ciExpectations[got]
				assert.True(t, ok, "every matrix image family must have a ciExpectations entry")
			}
		})
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
