package node_stats

import (
	"encoding/json"
	"testing"
)

func TestXPackZGCMapping(t *testing.T) {
	// 测试 X-Pack 模式下的 ZGC 映射
	zgcData := `{
		"nodes": {
			"zgc_node": {
				"name": "test-zgc-node",
				"transport_address": "10.3.8.12:9300",
				"jvm": {
					"mem": {
						"heap_used_in_bytes": 2881486848,
						"heap_used_percent": 33,
						"heap_max_in_bytes": 8589934592
					},
					"gc": {
						"collectors": {
							"ZGC Cycles": {
								"collection_count": 28349,
								"collection_time_in_millis": 45101619
							},
							"ZGC Pauses": {
								"collection_count": 85061,
								"collection_time_in_millis": 1004
							}
						}
					}
				},
				"indices": {
					"docs": {"count": 1000},
					"store": {"size_in_bytes": 1000000},
					"indexing": {"index_total": 100, "index_time_in_millis": 5000, "throttle_time_in_millis": 0},
					"search": {"query_total": 50, "query_time_in_millis": 2000},
					"query_cache": {"memory_size_in_bytes": 100000, "hit_count": 10, "miss_count": 5, "evictions": 0},
					"fielddata": {"memory_size_in_bytes": 50000, "evictions": 0},
					"segments": {"count": 10, "memory_in_bytes": 200000, "terms_memory_in_bytes": 50000, "stored_fields_memory_in_bytes": 30000, "term_vectors_memory_in_bytes": 0, "norms_memory_in_bytes": 1000, "points_memory_in_bytes": 5000, "doc_values_memory_in_bytes": 10000, "index_writer_memory_in_bytes": 20000, "version_map_memory_in_bytes": 5000, "fixed_bit_set_memory_in_bytes": 1000},
					"request_cache": {"memory_size_in_bytes": 25000, "evictions": 0, "hit_count": 15, "miss_count": 3}
				},
				"os": {
					"cpu": {}
				},
				"process": {
					"open_file_descriptors": 100,
					"max_file_descriptors": 1000,
					"cpu": {"percent": 10}
				},
				"thread_pool": {
					"write": {"threads": 4, "queue": 0, "rejected": 0},
					"generic": {"threads": 4, "queue": 0, "rejected": 0},
					"get": {"threads": 1, "queue": 0, "rejected": 0},
					"management": {"threads": 5, "queue": 0, "rejected": 0},
					"search": {"threads": 13, "queue": 0, "rejected": 0}
				},
				"fs": {
					"total": {
						"total_in_bytes": 100000000,
						"free_in_bytes": 50000000,
						"available_in_bytes": 45000000
					}
				}
			}
		}
	}`

	var nodeData nodesStruct
	err := json.Unmarshal([]byte(zgcData), &nodeData)
	if err != nil {
		t.Fatalf("Failed to unmarshal test data: %v", err)
	}

	for id, node := range nodeData.Nodes {
		t.Logf("Processing X-Pack ZGC node %s", id)
		result, err := schemaXpack.Apply(node)
		if err != nil {
			t.Errorf("X-Pack schema apply failed: %v", err)
		} else {
			t.Log("X-Pack schema apply succeeded")
			// 检查 GC 收集器
			if jvm, ok := result["jvm"].(map[string]interface{}); ok {
				if gc, ok := jvm["gc"].(map[string]interface{}); ok {
					if collectors, ok := gc["collectors"].(map[string]interface{}); ok {
						t.Logf("ZGC Node collectors: %v", collectors)
						// 验证 ZGC 收集器存在
						if _, hasZGCCycles := collectors["zgc_cycles"]; hasZGCCycles {
							t.Log("✓ ZGC Cycles collector found")
						}
						if _, hasZGCPauses := collectors["zgc_pauses"]; hasZGCPauses {
							t.Log("✓ ZGC Pauses collector found")
						}
					}
				}
			}
		}
	}
}

func TestXPackTraditionalGCMapping(t *testing.T) {
	// 测试 X-Pack 模式下的传统 GC 映射
	traditionalData := `{
		"nodes": {
			"traditional_node": {
				"name": "test-traditional-node",
				"transport_address": "10.3.10.126:9300",
				"jvm": {
					"mem": {
						"heap_used_in_bytes": 2594955392,
						"heap_used_percent": 30,
						"heap_max_in_bytes": 8589934592
					},
					"gc": {
						"collectors": {
							"young": {
								"collection_count": 475449,
								"collection_time_in_millis": 5713411
							},
							"old": {
								"collection_count": 0,
								"collection_time_in_millis": 0
							}
						}
					}
				},
				"indices": {
					"docs": {"count": 1000},
					"store": {"size_in_bytes": 1000000},
					"indexing": {"index_total": 100, "index_time_in_millis": 5000, "throttle_time_in_millis": 0},
					"search": {"query_total": 50, "query_time_in_millis": 2000},
					"query_cache": {"memory_size_in_bytes": 100000, "hit_count": 10, "miss_count": 5, "evictions": 0},
					"fielddata": {"memory_size_in_bytes": 50000, "evictions": 0},
					"segments": {"count": 10, "memory_in_bytes": 200000, "terms_memory_in_bytes": 50000, "stored_fields_memory_in_bytes": 30000, "term_vectors_memory_in_bytes": 0, "norms_memory_in_bytes": 1000, "points_memory_in_bytes": 5000, "doc_values_memory_in_bytes": 10000, "index_writer_memory_in_bytes": 20000, "version_map_memory_in_bytes": 5000, "fixed_bit_set_memory_in_bytes": 1000},
					"request_cache": {"memory_size_in_bytes": 25000, "evictions": 0, "hit_count": 15, "miss_count": 3}
				},
				"os": {
					"cpu": {}
				},
				"process": {
					"open_file_descriptors": 100,
					"max_file_descriptors": 1000,
					"cpu": {"percent": 10}
				},
				"thread_pool": {
					"write": {"threads": 4, "queue": 0, "rejected": 0},
					"generic": {"threads": 4, "queue": 0, "rejected": 0},
					"get": {"threads": 1, "queue": 0, "rejected": 0},
					"management": {"threads": 5, "queue": 0, "rejected": 0},
					"search": {"threads": 13, "queue": 0, "rejected": 0}
				},
				"fs": {
					"total": {
						"total_in_bytes": 100000000,
						"free_in_bytes": 50000000,
						"available_in_bytes": 45000000
					}
				}
			}
		}
	}`

	var nodeData nodesStruct
	err := json.Unmarshal([]byte(traditionalData), &nodeData)
	if err != nil {
		t.Fatalf("Failed to unmarshal test data: %v", err)
	}

	for id, node := range nodeData.Nodes {
		t.Logf("Processing X-Pack Traditional GC node %s", id)
		result, err := schemaXpack.Apply(node)
		if err != nil {
			t.Errorf("X-Pack schema apply failed: %v", err)
		} else {
			t.Log("X-Pack schema apply succeeded")
			// 检查 GC 收集器
			if jvm, ok := result["jvm"].(map[string]interface{}); ok {
				if gc, ok := jvm["gc"].(map[string]interface{}); ok {
					if collectors, ok := gc["collectors"].(map[string]interface{}); ok {
						t.Logf("Traditional GC Node collectors: %v", collectors)
						// 验证传统 GC 收集器存在
						if _, hasYoung := collectors["young"]; hasYoung {
							t.Log("✓ Young collector found")
						}
						if _, hasOld := collectors["old"]; hasOld {
							t.Log("✓ Old collector found")
						}
					}
				}
			}
		}
	}
}
