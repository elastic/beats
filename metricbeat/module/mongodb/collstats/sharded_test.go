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
