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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestFlattenAggregationResult_NoStorageStats(t *testing.T) {
	input := map[string]interface{}{
		"size": float64(10),
	}
	out := flattenAggregationResult(input)
	sizef, ok := out["size"].(float64)
	if ok && sizef != 10 {
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

	scaleFactorInt, ok := out["scaleFactor"].(int)
	if ok && scaleFactorInt != 1 {
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
	countInt, ok := out["count"].(int64)
	if ok && countInt != 999 {
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
	assert.InDelta(t, float64(2048), result["size"], 0)           // Unchanged
	assert.InDelta(t, float64(4096), result["storageSize"], 0)    // Unchanged
	assert.InDelta(t, float64(1024), result["totalIndexSize"], 0) // Unchanged
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
	assert.InDelta(t, float64(3000), result["size"], 0) // Unchanged
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
				assert.InDelta(t, tt.expectedSize, size, 0)
			}

			// Verify shard count and shard metadata cleanup
			if len(tt.shardResults) > 0 {
				// Verify shard count
				shardCount, exists := result["shardCount"]
				assert.True(t, exists)
				assert.Equal(t, len(tt.shardResults), shardCount)
				// No shards breakdown should be present
				_, exists = result["shards"]
				assert.False(t, exists)
				_, exists = result["shard"]
				assert.False(t, exists)
				_, exists = result["host"]
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
			name:     "float32",
			input:    float32(1.5),
			expected: 1.5,
			success:  true,
		},
		{
			name:     "int",
			input:    int(123),
			expected: 123.0,
			success:  true,
		},
		{
			name:     "int8",
			input:    int8(12),
			expected: 12.0,
			success:  true,
		},
		{
			name:     "int16",
			input:    int16(123),
			expected: 123.0,
			success:  true,
		},
		{
			name:     "uint",
			input:    uint(123),
			expected: 123.0,
			success:  true,
		},
		{
			name:     "uint8",
			input:    uint8(123),
			expected: 123.0,
			success:  true,
		},
		{
			name:     "uint16",
			input:    uint16(123),
			expected: 123.0,
			success:  true,
		},
		{
			name:     "uint32",
			input:    uint32(123),
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
				assert.InDelta(t, tt.expected, result, 0)
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
	assert.InDelta(t, float64(87.5), avgObjSize, 0)
}

func TestMergeShardedCollStats_SumsFreeStorageSize(t *testing.T) {
	shardResults := []map[string]interface{}{
		{
			"shard":           "shard01",
			"count":           int64(10),
			"freeStorageSize": float64(128),
		},
		{
			"shard":           "shard02",
			"count":           int64(20),
			"freeStorageSize": float64(256),
		},
	}

	result, err := mergeShardedCollStats(shardResults)
	assert.NoError(t, err)
	assert.InDelta(t, float64(384), result["freeStorageSize"], 0)
}

func TestMergeShardedCollStats_ZeroDocsSetsAvgObjSizeZero(t *testing.T) {
	// When no documents exist across shards, avgObjSize must be 0 (mongosh parity)
	// even if a shard reports a stale avgObjSize.
	shardResults := []map[string]interface{}{
		{"shard": "shard01", "count": int64(0), "avgObjSize": float64(50)},
		{"shard": "shard02", "count": int64(0), "avgObjSize": float64(75)},
	}

	result, err := mergeShardedCollStats(shardResults)
	assert.NoError(t, err)
	assert.Equal(t, 0, result["avgObjSize"])
}

func TestMergeShardedCollStats_WeightedAvgSkipsShardsMissingCount(t *testing.T) {
	// A shard missing count must not contribute to the weighted avgObjSize.
	shardResults := []map[string]interface{}{
		{"shard": "shard01", "count": int64(100), "avgObjSize": float64(40)},
		// no count -> excluded from weighting AND from totalDocCount
		{"shard": "shard02", "avgObjSize": float64(1000)},
	}

	result, err := mergeShardedCollStats(shardResults)
	assert.NoError(t, err)
	// totalDocCount = 100, weightedSum = 40*100 -> 4000/100 = 40
	assert.InDelta(t, float64(40), result["avgObjSize"].(float64), 0.0001) //nolint:errcheck // safe
}

func TestMergeShardedCollStats_MaxFieldsTakeMaximum(t *testing.T) {
	shardResults := []map[string]interface{}{
		{"shard": "shard01", "count": int64(1), "max": int64(100), "maxSize": float64(2048)},
		{"shard": "shard02", "count": int64(1), "max": int64(500), "maxSize": float64(1024)},
	}

	result, err := mergeShardedCollStats(shardResults)
	assert.NoError(t, err)
	assert.InDelta(t, float64(500), result["max"].(float64), 0)      //nolint:errcheck // safe
	assert.InDelta(t, float64(2048), result["maxSize"].(float64), 0) //nolint:errcheck // safe
}

func TestMergeShardedCollStats_RemovesLocalTimeMetadata(t *testing.T) {
	shardResults := []map[string]interface{}{
		{"shard": "shard01", "host": "h1", "localTime": "t1", "count": int64(5)},
	}

	result, err := mergeShardedCollStats(shardResults)
	assert.NoError(t, err)
	_, hasLocalTime := result["localTime"]
	assert.False(t, hasLocalTime)
}

func TestMergeShardedCollStats_IgnoresNonNumericSummableFields(t *testing.T) {
	// A non-numeric value for a summable field must be skipped, not crash or coerce.
	shardResults := []map[string]interface{}{
		{"shard": "shard01", "count": "not-a-number", "size": float64(10)},
		{"shard": "shard02", "count": int64(7), "size": float64(20)},
	}

	result, err := mergeShardedCollStats(shardResults)
	assert.NoError(t, err)
	// Only the numeric count contributes.
	assert.Equal(t, int64(7), result["count"].(int64))          //nolint:errcheck // safe
	assert.InDelta(t, float64(30), result["size"].(float64), 0) //nolint:errcheck // safe
}

func TestHasShardMetadata(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected bool
	}{
		{name: "nil", input: nil, expected: false},
		{name: "no metadata", input: map[string]interface{}{"count": int64(1)}, expected: false},
		{name: "has shard", input: map[string]interface{}{"shard": "s1"}, expected: true},
		// "host" alone is NOT a sharding signal: $collStats always emits a
		// top-level host (the serving node), including on standalone/replica set.
		{name: "host only (standalone $collStats)", input: map[string]interface{}{"host": "h1"}, expected: false},
		{name: "has both", input: map[string]interface{}{"shard": "s1", "host": "h1"}, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, hasShardMetadata(tt.input))
		})
	}
}

func TestFlattenAggregationResult_NilInput(t *testing.T) {
	assert.Nil(t, flattenAggregationResult(nil))
}

func TestFlattenAggregationResult_StorageStatsWrongType(t *testing.T) {
	// storageStats present but not a map must be returned unchanged (no panic).
	input := map[string]interface{}{
		"storageStats": "unexpected-string",
		"size":         float64(5),
	}
	out := flattenAggregationResult(input)
	assert.Equal(t, "unexpected-string", out["storageStats"])
	assert.InDelta(t, float64(5), out["size"].(float64), 0) //nolint:errcheck // safe
	// Nothing lifted, so scaleFactor default path is not applied for this branch.
	_, hasScaleFactor := out["scaleFactor"]
	assert.False(t, hasScaleFactor)
}

func TestFlattenAggregationResult_DefaultsScaleFactorWhenAbsent(t *testing.T) {
	// storageStats present without scaleFactor -> default to 1.
	input := map[string]interface{}{
		"storageStats": map[string]interface{}{
			"size": float64(100),
		},
	}
	out := flattenAggregationResult(input)
	assert.Equal(t, 1, out["scaleFactor"])
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
		// Major-only strings: missing minor/patch must be treated as 0, not as "equal so far → true".
		{
			name:     "major only below boundary",
			current:  "6",
			target:   "6.2.0",
			expected: false, // "6" == "6.0.0", which is < 6.2.0
		},
		{
			name:     "major only above boundary",
			current:  "7",
			target:   "6.2.0",
			expected: true,
		},
		// Major.minor-only strings: missing patch treated as 0.
		{
			name:     "major.minor below boundary",
			current:  "6.1",
			target:   "6.2.0",
			expected: false,
		},
		{
			name:     "major.minor at boundary",
			current:  "6.2",
			target:   "6.2.0",
			expected: true, // "6.2" == "6.2.0"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVersionAtLeast(tt.current, tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCacheMongoVersion_RetryOnError(t *testing.T) {
	callCount := 0
	ms := &Metricset{
		versionGetter: func(_ *mongo.Client) (string, error) {
			callCount++
			if callCount == 1 {
				return "", errors.New("temporary connection error")
			}
			return "7.0.0", nil
		},
	}

	// First call: getter fails → version must not be cached so next fetch retries.
	ms.cacheMongoVersion(nil)
	assert.Empty(t, ms.mongoVersion, "version must not be cached on error")
	assert.Equal(t, 1, callCount, "getter called once")

	// Second call: getter succeeds → version is now cached.
	ms.cacheMongoVersion(nil)
	assert.Equal(t, "7.0.0", ms.mongoVersion, "version cached after successful retry")
	assert.Equal(t, 2, callCount, "getter called a second time")

	// Third call: already cached → getter not called again.
	ms.cacheMongoVersion(nil)
	assert.Equal(t, "7.0.0", ms.mongoVersion, "cached version unchanged")
	assert.Equal(t, 2, callCount, "getter not called when version already cached")
}

func TestCacheMongoVersion_EmptyStringNotCached(t *testing.T) {
	// If the getter returns an empty version with no error (e.g. malformed server
	// response), the empty string must not be stored — next fetch should retry.
	callCount := 0
	ms := &Metricset{
		versionGetter: func(_ *mongo.Client) (string, error) {
			callCount++
			if callCount == 1 {
				return "", nil // empty but no error
			}
			return "6.0.0", nil
		},
	}

	ms.cacheMongoVersion(nil)
	assert.Empty(t, ms.mongoVersion, "empty version must not be cached")
	assert.Equal(t, 1, callCount)

	ms.cacheMongoVersion(nil)
	assert.Equal(t, "6.0.0", ms.mongoVersion, "real version cached on retry")
	assert.Equal(t, 2, callCount)
}

func TestCacheMongoVersion_PermanentFailureUsesLegacyPath(t *testing.T) {
	// A getter that always fails must leave mongoVersion as "" so every Fetch
	// retries and always falls through to the legacy collStats command path.
	ms := &Metricset{
		versionGetter: func(_ *mongo.Client) (string, error) {
			return "", errors.New("connection refused")
		},
	}

	for i := 0; i < 3; i++ {
		ms.cacheMongoVersion(nil)
		assert.Empty(t, ms.mongoVersion, "version must remain uncached after failure #%d", i+1)
		assert.False(t, isVersionAtLeast(ms.mongoVersion, "6.2.0"), "must use legacy path when version unknown")
	}
}
