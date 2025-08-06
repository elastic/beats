package node_stats

import (
	"encoding/json"
	"testing"
)

func TestMixedGCMapping(t *testing.T) {
	// 测试混合场景：传统 GC 和 ZGC 节点
	mixedData := `{
		"nodes": {
			"traditional_node": {
				"name": "traditional-gc-node",
				"jvm": {
					"mem": {
						"pools": {
							"young": {
								"used_in_bytes": 583008256,
								"max_in_bytes": 0,
								"peak_used_in_bytes": 5125439488,
								"peak_max_in_bytes": 0
							},
							"old": {
								"used_in_bytes": 1966625784,
								"max_in_bytes": 8589934592,
								"peak_used_in_bytes": 5412663288,
								"peak_max_in_bytes": 8589934592
							},
							"survivor": {
								"used_in_bytes": 45321352,
								"max_in_bytes": 0,
								"peak_used_in_bytes": 599785472,
								"peak_max_in_bytes": 0
							}
						}
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
					"docs": {"count": 1000, "deleted": 10},
					"store": {"size_in_bytes": 1000000},
					"segments": {"count": 50, "memory_in_bytes": 5000000}
				},
				"fs": {
					"total": {
						"total_in_bytes": 100000000,
						"free_in_bytes": 50000000,
						"available_in_bytes": 45000000
					}
				}
			},
			"zgc_node": {
				"name": "zgc-node",
				"jvm": {
					"mem": {
						"pools": {}
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
					"docs": {"count": 1000, "deleted": 10},
					"store": {"size_in_bytes": 1000000},
					"segments": {"count": 50, "memory_in_bytes": 5000000}
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
	err := json.Unmarshal([]byte(mixedData), &nodeData)
	if err != nil {
		t.Fatalf("Failed to unmarshal test data: %v", err)
	}

	for id, node := range nodeData.Nodes {
		t.Logf("Processing node %s", id)
		result, err := schema.Apply(node)
		if err != nil {
			t.Errorf("Schema apply failed for node %s: %v", id, err)
		} else {
			t.Logf("Schema apply succeeded for node %s", id)
			// 检查 GC 收集器
			if gc, ok := result["jvm"].(map[string]interface{}); ok {
				if collectors, ok := gc["gc"].(map[string]interface{}); ok {
					if collectorData, ok := collectors["collectors"].(map[string]interface{}); ok {
						t.Logf("Node %s collectors: %v", id, collectorData)
					}
				}
			}
		}
	}
}
