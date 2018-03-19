package jmx

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestEventMapper(t *testing.T) {
	absPath, err := filepath.Abs("./_meta/test")

	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	jolokiaResponse, err := ioutil.ReadFile(absPath + "/jolokia_response.json")

	assert.Nil(t, err)

	var mapping = map[string]string{
		"java.lang:type=Runtime_Uptime":                                            "uptime",
		"java.lang:type=GarbageCollector,name=ConcurrentMarkSweep_CollectionTime":  "gc.cms_collection_time",
		"java.lang:type=GarbageCollector,name=ConcurrentMarkSweep_CollectionCount": "gc.cms_collection_count",
		"java.lang:type=Memory_HeapMemoryUsage":                                    "memory.heap_usage",
		"java.lang:type=Memory_NonHeapMemoryUsage":                                 "memory.non_heap_usage",
		"org.springframework.boot:type=Endpoint,name=metricsEndpoint_Metrics":      "metrics",
	}

	event, err := eventMapping(jolokiaResponse, mapping)
	assert.Nil(t, err)

	expected := common.MapStr{
		"uptime": float64(47283),
		"gc": common.MapStr{
			"cms_collection_time":  float64(53),
			"cms_collection_count": float64(1),
		},
		"memory": common.MapStr{
			"heap_usage": map[string]interface{}{
				"init":      float64(1073741824),
				"committed": float64(1037959168),
				"max":       float64(1037959168),
				"used":      float64(227420472),
			},
			"non_heap_usage": map[string]interface{}{
				"init":      float64(2555904),
				"committed": float64(53477376),
				"max":       float64(-1),
				"used":      float64(50519768),
			},
		},
		"metrics": map[string]interface{}{
			"atomikos_nbTransactions": float64(0),
			"classes":                 float64(18857),
			"classes_loaded":          float64(19127),
			"classes_unloaded":        float64(270),
		},
	}

	assert.Equal(t, expected, event)
}
