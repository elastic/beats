package node_stats

import (
	"encoding/json"
	"testing"
)

func TestZGCMapping(t *testing.T) {
	// 测试 ZGC 映射
	zgcData := `{
		"nodes": {
			"test_node": {
				"name": "test-node-zgc",
				"jvm": {
					"mem": {
						"pools": {}
					},
					"gc": {
						"collectors": {
							"ZGC Cycles": {
								"collection_count": 100,
								"collection_time_in_millis": 1000
							},
							"ZGC Pauses": {
								"collection_count": 200,
								"collection_time_in_millis": 50
							}
						}
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
		t.Logf("Processing node %s", id)
		_, err := schema.Apply(node)
		if err != nil {
			t.Errorf("Schema apply failed: %v", err)
		} else {
			t.Log("Schema apply succeeded")
		}
	}
}

func TestTraditionalGCMapping(t *testing.T) {
	// 测试传统 GC 映射
	traditionalData := `{
		"nodes": {
			"test_node": {
				"name": "test-node-traditional",
				"jvm": {
					"mem": {
						"pools": {
							"young": {
								"used_in_bytes": 100000,
								"max_in_bytes": 0,
								"peak_used_in_bytes": 200000,
								"peak_max_in_bytes": 0
							},
							"old": {
								"used_in_bytes": 500000,
								"max_in_bytes": 1000000,
								"peak_used_in_bytes": 600000,
								"peak_max_in_bytes": 1000000
							}
						}
					},
					"gc": {
						"collectors": {
							"young": {
								"collection_count": 100,
								"collection_time_in_millis": 1000
							},
							"old": {
								"collection_count": 10,
								"collection_time_in_millis": 500
							}
						}
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
		t.Logf("Processing node %s", id)
		_, err := schema.Apply(node)
		if err != nil {
			t.Errorf("Schema apply failed: %v", err)
		} else {
			t.Log("Schema apply succeeded")
		}
	}
}
