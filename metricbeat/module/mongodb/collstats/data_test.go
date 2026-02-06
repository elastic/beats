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

//go:build !requirefips && integration

package collstats

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestEventMapping(t *testing.T) {
	content, err := os.ReadFile("./_meta/test/input.json")
	assert.NoError(t, err)

	data := mapstr.M{}
	err = json.Unmarshal(content, &data)
	if err != nil {
		t.Fatal(err)
	}

	event, _ := eventMapping("unit.test", data)

	assert.Equal(t, event["total"].(mapstr.M)["count"], float64(1)) //nolint:errcheck // safe
}

func TestEventMappingOptionalFields(t *testing.T) {
	// Build a data map emulating flattened aggregation output including optional fields
	data := mapstr.M{
		"total":     mapstr.M{"count": 5, "time": 123},
		"readLock":  mapstr.M{"count": 2, "time": 10},
		"writeLock": mapstr.M{"count": 3, "time": 20},
		"queries":   mapstr.M{"count": 1, "time": 2},
		"getmore":   mapstr.M{"count": 0, "time": 0},
		"insert":    mapstr.M{"count": 0, "time": 0},
		"update":    mapstr.M{"count": 0, "time": 0},
		"remove":    mapstr.M{"count": 0, "time": 0},
		"commands":  mapstr.M{"count": 0, "time": 0},
		"stats": mapstr.M{
			"size":            1000,
			"count":           5,
			"avgObjSize":      200,
			"storageSize":     2048,
			"totalIndexSize":  512,
			"totalSize":       2560,
			"max":             100000,
			"nindexes":        2,
			"numOrphanDocs":   1,
			"shardCount":      2,
			"freeStorageSize": 4096,
			"capped":          false,
			"scaleFactor":     1024,
			// shards breakdown and indexSizes omitted
		},
	}

	event, err := eventMapping("dbX.collY", data)
	assert.NoError(t, err)

	stats := event["stats"].(mapstr.M)                    //nolint:errcheck // safe
	assert.Equal(t, int(1), stats["numOrphanDocs"].(int)) //nolint:errcheck // safe
	assert.Equal(t, int(2), stats["shardCount"].(int))    //nolint:errcheck // safe
	assert.Equal(t, 4096, stats["freeStorageSize"].(int)) //nolint:errcheck // safe
	assert.Equal(t, false, stats["capped"].(bool))        //nolint:errcheck // safe
	assert.Equal(t, 1024, stats["scaleFactor"].(int))     //nolint:errcheck // safe
	_, hasIndexSizes := stats["indexSizes"]
	assert.False(t, hasIndexSizes)
}

func TestMergeShardedCollStats_WeightedAndIndexMerge(t *testing.T) {
	shard1 := map[string]interface{}{
		"count":       int64(10),
		"size":        float64(1000),
		"storageSize": float64(2000),
		"avgObjSize":  float64(100),
		"shard":       "shard1",
		"host":        "host1",
	}
	shard2 := map[string]interface{}{
		"count":       int64(20),
		"size":        float64(3000),
		"storageSize": float64(4000),
		"avgObjSize":  float64(150),
		"shard":       "shard2",
		"host":        "host2",
	}
	merged, err := mergeShardedCollStats([]map[string]interface{}{shard1, shard2})
	assert.NoError(t, err)
	// Summed fields
	assert.Equal(t, int64(30), merged["count"].(int64))      //nolint:errcheck // safe
	assert.Equal(t, float64(4000), merged["size"].(float64)) //nolint:errcheck // safe
	// Weighted avg: (100*10 + 150*20)/30 = (1000 + 3000)/30 = 133.333...
	assert.InDelta(t, 133.33, merged["avgObjSize"].(float64), 0.01) //nolint:errcheck // safe
	// No indexSizes collected
	_, hasIdx := merged["indexSizes"]
	assert.False(t, hasIdx)
	// Shard metadata only: shardCount present, no shards breakdown
	assert.Equal(t, 2, merged["shardCount"].(int)) //nolint:errcheck // safe
	_, hasShards := merged["shards"]
	assert.False(t, hasShards)
}

func TestFlattenAggregationResult(t *testing.T) {
	input := map[string]interface{}{
		"storageStats": map[string]interface{}{
			"size":           1000,
			"count":          5,
			"avgObjSize":     200,
			"storageSize":    2048,
			"totalIndexSize": 512,
			"totalSize":      2560,
			"scaleFactor":    1024,
		},
	}
	flat := flattenAggregationResult(input)
	// All expected top-level lifts
	expectedKeys := []string{"size", "count", "avgObjSize", "storageSize", "totalIndexSize", "totalSize", "scaleFactor"}
	for _, k := range expectedKeys {
		assert.Contains(t, flat, k)
	}
	assert.Equal(t, 1024, flat["scaleFactor"])
}
