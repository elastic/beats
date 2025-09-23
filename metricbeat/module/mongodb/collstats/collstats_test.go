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

//go:build !requirefips

package collstats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlattenAggregationResult_NoStorageStats(t *testing.T) {
	input := map[string]interface{}{
		"size": float64(10),
	}
	out := flattenAggregationResult(input)
	if out["size"].(float64) != 10 {
		t.Fatalf("expected size 10, got %v", out["size"])
	}
}

func TestFlattenAggregationResult_WithStorageStats(t *testing.T) {
	input := map[string]interface{}{
		"storageStats": map[string]interface{}{
			"size":            float64(1000),
			"count":           int64(50),
			"avgObjSize":      float64(20),
			"storageSize":     float64(2000),
			"totalIndexSize":  float64(3000),
			"totalSize":       float64(5000),
			"nindexes":        int64(3),
			"freeStorageSize": float64(10),
			"capped":          false,
			"numOrphanDocs":   int64(0),
		},
	}
	out := flattenAggregationResult(input)

	// Ensure fields were lifted
	if _, ok := out["size"]; !ok {
		t.Fatalf("size not lifted")
	}
	if _, ok := out["count"]; !ok {
		t.Fatalf("count not lifted")
	}
	if _, ok := out["indexSizes"]; ok {
		t.Fatalf("indexSizes should not be lifted")
	}
	if _, ok := out["freeStorageSize"]; !ok {
		t.Fatalf("freeStorageSize not lifted")
	}
	if _, ok := out["capped"]; !ok {
		t.Fatalf("capped not lifted")
	}
	if _, ok := out["numOrphanDocs"]; !ok {
		t.Fatalf("numOrphanDocs not lifted")
	}
	if out["scaleFactor"].(int) != 1 {
		t.Fatalf("expected scaleFactor=1, got %v", out["scaleFactor"])
	}
}

func TestFlattenAggregationResult_DoesNotOverrideExisting(t *testing.T) {
	input := map[string]interface{}{
		"count": int64(999), // pre-existing value should not be overridden
		"storageStats": map[string]interface{}{
			"count": int64(50),
		},
	}
	out := flattenAggregationResult(input)
	if out["count"].(int64) != 999 {
		t.Fatalf("expected existing count 999 preserved, got %v", out["count"])
	}
}

func TestApplyOptionsToStats_NoClientRescale(t *testing.T) {
	metricset := &Metricset{options: CollStatsOptions{Scale: 1024}}
	// Simulate server already applied scale (so values are post-scale). We expect applyOptionsToStats to leave them untouched.
	stats := map[string]interface{}{
		"size":           float64(2048), // Already scaled KB value
		"storageSize":    float64(4096),
		"totalIndexSize": float64(1024),
		"count":          int64(1000),
	}
	result, err := metricset.applyOptionsToStats(stats)
	assert.NoError(t, err)
	assert.Equal(t, float64(2048), result["size"])           // Unchanged
	assert.Equal(t, float64(4096), result["storageSize"])    // Unchanged
	assert.Equal(t, float64(1024), result["totalIndexSize"]) // Unchanged
	// indexSizes.* are not collected currently
	_, hasIdx := result["indexSizes"]
	assert.False(t, hasIdx)
}

func TestApplyOptionsToStats_NoShardRescale(t *testing.T) {
	metricset := &Metricset{options: CollStatsOptions{Scale: 1024}}
	stats := map[string]interface{}{
		"size": float64(3000), // Already scaled total (KB)
		// shards.* breakdown not collected
	}
	result, err := metricset.applyOptionsToStats(stats)
	assert.NoError(t, err)
	assert.Equal(t, float64(3000), result["size"]) // Unchanged
	_, hasShards := result["shards"]
	assert.False(t, hasShards)
}

func TestApplyOptionsToStats_NilStats(t *testing.T) {
	metricset := &Metricset{
		options: CollStatsOptions{
			Scale: 1024,
		},
	}

	result, err := metricset.applyOptionsToStats(nil)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestMergeShardedCollStats(t *testing.T) {
	tests := []struct {
		name          string
		shardResults  []map[string]interface{}
		expectedCount int64
		expectedSize  float64
		expectedErr   bool
	}{
		{
			name: "single shard",
			shardResults: []map[string]interface{}{
				{
					"ns":          "test.collection",
					"shard":       "shard01",
					"host":        "shard01:27017",
					"count":       int64(1000),
					"size":        float64(50000),
					"storageSize": float64(60000),
					"avgObjSize":  float64(50),
				},
			},
			expectedCount: 1000,
			expectedSize:  50000,
			expectedErr:   false,
		},
		{
			name: "multiple shards with summable fields",
			shardResults: []map[string]interface{}{
				{
					"ns":          "test.collection",
					"shard":       "shard01",
					"host":        "shard01:27017",
					"count":       int64(1000),
					"size":        float64(50000),
					"storageSize": float64(60000),
					"avgObjSize":  float64(50),
				},
				{
					"ns":          "test.collection",
					"shard":       "shard02",
					"host":        "shard02:27017",
					"count":       int64(2000),
					"size":        float64(120000),
					"storageSize": float64(130000),
					"avgObjSize":  float64(60),
				},
			},
			expectedCount: 3000,
			expectedSize:  170000,
			expectedErr:   false,
		},
		{
			name: "shards with index sizes (ignored)",
			shardResults: []map[string]interface{}{
				{
					"ns":    "test.collection",
					"shard": "shard01",
					"host":  "shard01:27017",
					"count": int64(1000),
					"size":  float64(50000),
					// indexSizes will be ignored
				},
				{
					"ns":    "test.collection",
					"shard": "shard02",
					"host":  "shard02:27017",
					"count": int64(2000),
					"size":  float64(120000),
					// indexSizes will be ignored
				},
			},
			expectedCount: 3000,
			expectedSize:  170000,
			expectedErr:   false,
		},
		{
			name:          "empty shard results",
			shardResults:  []map[string]interface{}{},
			expectedCount: 0,
			expectedSize:  0,
			expectedErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mergeShardedCollStats(tt.shardResults)

			if tt.expectedErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)

			// Check merged counts
			if count, exists := result["count"]; exists {
				assert.Equal(t, tt.expectedCount, count)
			}

			// Check merged sizes
			if size, exists := result["size"]; exists {
				assert.Equal(t, tt.expectedSize, size)
			}

			// For multi-shard tests, verify shardCount reported but no shards breakdown
			if len(tt.shardResults) > 1 {
				// Verify shard count
				shardCount, exists := result["shardCount"]
				assert.True(t, exists)
				assert.Equal(t, len(tt.shardResults), shardCount)
				// No shards breakdown should be present
				_, exists = result["shards"]
				assert.False(t, exists)
			}
		})
	}
}

func TestConvertToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
		success  bool
	}{
		{
			name:     "float64",
			input:    float64(123.45),
			expected: 123.45,
			success:  true,
		},
		{
			name:     "int64",
			input:    int64(123),
			expected: 123.0,
			success:  true,
		},
		{
			name:     "int32",
			input:    int32(123),
			expected: 123.0,
			success:  true,
		},
		{
			name:     "uint64",
			input:    uint64(123),
			expected: 123.0,
			success:  true,
		},
		{
			name:     "string",
			input:    "123",
			expected: 0,
			success:  false,
		},
		{
			name:     "nil",
			input:    nil,
			expected: 0,
			success:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, success := convertToFloat64(tt.input)
			assert.Equal(t, tt.success, success)
			if tt.success {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestMergeShardedCollStats_WeightedAverages(t *testing.T) {
	shardResults := []map[string]interface{}{
		{
			"shard":      "shard01",
			"count":      int64(1000),
			"avgObjSize": float64(50), // 1000 docs * 50 bytes = 50000 total
		},
		{
			"shard":      "shard02",
			"count":      int64(3000),
			"avgObjSize": float64(100), // 3000 docs * 100 bytes = 300000 total
		},
	}

	result, err := mergeShardedCollStats(shardResults)
	assert.NoError(t, err)

	// Total: 4000 docs, 350000 total bytes
	// Expected average: 350000 / 4000 = 87.5
	avgObjSize, exists := result["avgObjSize"]
	assert.True(t, exists)
	assert.Equal(t, float64(87.5), avgObjSize)
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected []int
	}{
		{
			name:     "simple version",
			version:  "6.2.0",
			expected: []int{6, 2, 0},
		},
		{
			name:     "version with rc",
			version:  "6.2.0-rc1",
			expected: []int{6, 2, 0},
		},
		{
			name:     "version with build metadata",
			version:  "7.0.1+build123",
			expected: []int{7, 0, 1},
		},
		{
			name:     "major.minor only",
			version:  "5.0",
			expected: []int{5, 0, 0},
		},
		{
			name:     "major only",
			version:  "4",
			expected: []int{4, 0, 0},
		},
		{
			name:     "version with extra parts",
			version:  "6.2.0.1.2",
			expected: []int{6, 2, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVersion(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsVersionAtLeast(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		target   string
		expected bool
	}{
		{
			name:     "exact match",
			current:  "6.2.0",
			target:   "6.2.0",
			expected: true,
		},
		{
			name:     "current newer major",
			current:  "7.0.0",
			target:   "6.2.0",
			expected: true,
		},
		{
			name:     "current newer minor",
			current:  "6.3.0",
			target:   "6.2.0",
			expected: true,
		},
		{
			name:     "current newer patch",
			current:  "6.2.1",
			target:   "6.2.0",
			expected: true,
		},
		{
			name:     "current older major",
			current:  "5.0.0",
			target:   "6.2.0",
			expected: false,
		},
		{
			name:     "current older minor",
			current:  "6.1.0",
			target:   "6.2.0",
			expected: false,
		},
		{
			name:     "current older patch",
			current:  "6.2.0",
			target:   "6.2.1",
			expected: false,
		},
		{
			name:     "unknown version",
			current:  "unknown",
			target:   "6.2.0",
			expected: false,
		},
		{
			name:     "empty version",
			current:  "",
			target:   "6.2.0",
			expected: false,
		},
		{
			name:     "version with rc suffix",
			current:  "6.2.0-rc1",
			target:   "6.2.0",
			expected: true,
		},
		{
			name:     "version with build metadata",
			current:  "7.0.0+build123",
			target:   "6.2.0",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVersionAtLeast(tt.current, tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}
