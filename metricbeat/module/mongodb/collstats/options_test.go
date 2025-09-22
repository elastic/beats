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
