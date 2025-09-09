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

import "testing"

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
			"size":           float64(1000),
			"count":          int64(50),
			"avgObjSize":     float64(20),
			"storageSize":    float64(2000),
			"totalIndexSize": float64(3000),
			"totalSize":      float64(5000),
			"nindexes":       int64(3),
			"indexSizes": map[string]interface{}{
				"_id_": float64(100),
			},
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
	if _, ok := out["indexSizes"]; !ok {
		t.Fatalf("indexSizes not lifted")
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
