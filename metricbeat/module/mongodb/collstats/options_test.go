// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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

func TestApplyOptionsToStats_NoClientRescale(t *testing.T) {
	metricset := &Metricset{options: CollStatsOptions{Scale: 1024}}
	// Simulate server already applied scale (so values are post-scale). We expect applyOptionsToStats to leave them untouched.
	stats := map[string]interface{}{
		"size":           float64(2048), // Already scaled KB value
		"storageSize":    float64(4096),
		"totalIndexSize": float64(1024),
		"count":          int64(1000),
		"indexSizes": map[string]interface{}{
			"_id_":       float64(512),
			"name_index": float64(256),
		},
	}
	result, err := metricset.applyOptionsToStats(stats)
	assert.NoError(t, err)
	assert.Equal(t, float64(2048), result["size"])           // Unchanged
	assert.Equal(t, float64(4096), result["storageSize"])    // Unchanged
	assert.Equal(t, float64(1024), result["totalIndexSize"]) // Unchanged
	idx := result["indexSizes"].(map[string]interface{})
	assert.Equal(t, float64(512), idx["_id_"])
	assert.Equal(t, float64(256), idx["name_index"])
}

func TestApplyOptionsToStats_NoShardRescale(t *testing.T) {
	metricset := &Metricset{options: CollStatsOptions{Scale: 1024}}
	stats := map[string]interface{}{
		"size": float64(3000), // Already scaled total (KB)
		"shards": []map[string]interface{}{
			{"shard": "shard01", "size": float64(1000)},
			{"shard": "shard02", "size": float64(2000)},
		},
	}
	result, err := metricset.applyOptionsToStats(stats)
	assert.NoError(t, err)
	assert.Equal(t, float64(3000), result["size"]) // Unchanged
	shards := result["shards"].([]map[string]interface{})
	assert.Equal(t, float64(1000), shards[0]["size"]) // Unchanged
	assert.Equal(t, float64(2000), shards[1]["size"]) // Unchanged
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
