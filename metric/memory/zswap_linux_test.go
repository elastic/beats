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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

func TestGetZswapDebugMetrics(t *testing.T) {
	// Create a temporary directory structure that mimics /sys/kernel/debug/zswap
	tmpDir := t.TempDir()
	zswapDir := filepath.Join(tmpDir, "sys", "kernel", "debug", "zswap")
	require.NoError(t, os.MkdirAll(zswapDir, 0755))

	// Write test data
	testData := map[string]string{
		"stored_pages":          "1109442",
		"pool_total_size":       "3095379968",
		"written_back_pages":    "2489374",
		"reject_compress_poor":  "1271198",
		"reject_compress_fail":  "5531019",
		"reject_kmemcache_fail": "0",
		"reject_alloc_fail":     "0",
		"reject_reclaim_fail":   "26833",
		"pool_limit_hit":        "8353",
	}

	for name, value := range testData {
		require.NoError(t, os.WriteFile(filepath.Join(zswapDir, name), []byte(value), 0644))
	}

	metrics := getZswapDebugMetrics(resolve.NewTestResolver(tmpDir))

	expected := ZswapDebugMetrics{
		StoredPages:         opt.UintWith(1109442),
		PoolTotalSize:       opt.UintWith(3095379968),
		WrittenBackPages:    opt.UintWith(2489374),
		RejectCompressPoor:  opt.UintWith(1271198),
		RejectCompressFail:  opt.UintWith(5531019),
		RejectKmemcacheFail: opt.UintWith(0),
		RejectAllocFail:     opt.UintWith(0),
		RejectReclaimFail:   opt.UintWith(26833),
		PoolLimitHit:        opt.UintWith(8353),
	}
	assert.Equal(t, expected, metrics)
}

func TestGetZswapDebugMetrics_MissingDirectory(t *testing.T) {
	// Test with a non-existent directory - should return empty metrics without error
	tmpDir := t.TempDir()
	metrics := getZswapDebugMetrics(resolve.NewTestResolver(tmpDir))

	// All fields should be empty
	assert.True(t, metrics.IsZero())
}

func TestGetZswapDebugMetrics_PartialFiles(t *testing.T) {
	// Create a temporary directory with only some of the metric files
	tmpDir := t.TempDir()
	zswapDir := filepath.Join(tmpDir, "sys", "kernel", "debug", "zswap")
	require.NoError(t, os.MkdirAll(zswapDir, 0755))

	// Only write some files
	require.NoError(t, os.WriteFile(filepath.Join(zswapDir, "stored_pages"), []byte("12345"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(zswapDir, "pool_total_size"), []byte("67890"), 0644))

	metrics := getZswapDebugMetrics(resolve.NewTestResolver(tmpDir))

	expected := ZswapDebugMetrics{
		StoredPages:   opt.UintWith(12345),
		PoolTotalSize: opt.UintWith(67890),
	}
	assert.Equal(t, expected, metrics)
}

func TestGetZswapDebugMetrics_InvalidData(t *testing.T) {
	// Create a temporary directory with invalid data
	tmpDir := t.TempDir()
	zswapDir := filepath.Join(tmpDir, "sys", "kernel", "debug", "zswap")
	require.NoError(t, os.MkdirAll(zswapDir, 0755))

	// Write invalid data
	require.NoError(t, os.WriteFile(filepath.Join(zswapDir, "stored_pages"), []byte("not_a_number"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(zswapDir, "pool_total_size"), []byte("12345"), 0644))

	metrics := getZswapDebugMetrics(resolve.NewTestResolver(tmpDir))

	// Invalid data results in empty value, valid data is still parsed
	expected := ZswapDebugMetrics{
		PoolTotalSize: opt.UintWith(12345),
	}
	assert.Equal(t, expected, metrics)
}

func TestZswapDebugMetricsIsZero(t *testing.T) {
	z := ZswapDebugMetrics{}
	assert.True(t, z.IsZero())

	z.StoredPages = readUintFromFile("/dev/null") // returns empty opt.Uint
	assert.True(t, z.IsZero())

	z = ZswapDebugMetrics{StoredPages: opt.UintWith(100)}
	assert.False(t, z.IsZero())
}

func TestGetZswapDebugMetrics_RealTestdata(t *testing.T) {
	// Test using actual files copied from /sys/kernel/debug/zswap
	metrics := getZswapDebugMetrics(resolve.NewTestResolver("./testdata"))

	expected := ZswapDebugMetrics{
		StoredPages:         opt.UintWith(17),
		PoolTotalSize:       opt.UintWith(147456),
		WrittenBackPages:    opt.UintWith(0),
		RejectCompressPoor:  opt.UintWith(0),
		RejectCompressFail:  opt.UintWith(0),
		RejectKmemcacheFail: opt.UintWith(0),
		RejectAllocFail:     opt.UintWith(0),
		RejectReclaimFail:   opt.UintWith(0),
		PoolLimitHit:        opt.UintWith(0),
	}
	assert.Equal(t, expected, metrics)
}
