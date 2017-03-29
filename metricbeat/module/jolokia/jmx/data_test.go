package jmx

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
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
	}

	event, err := eventMapping(jolokiaResponse, mapping)

	assert.Nil(t, err)
	assert.EqualValues(t, 47283, event["uptime"])
	assert.EqualValues(t, 53, event["gc"].(common.MapStr)["cms_collection_time"])
	assert.EqualValues(t, 1, event["gc"].(common.MapStr)["cms_collection_count"])
	assert.EqualValues(t, 1073741824, event["memory"].(common.MapStr)["heap_usage"].(map[string]interface{})["init"])
	assert.EqualValues(t, 1037959168, event["memory"].(common.MapStr)["heap_usage"].(map[string]interface{})["committed"])
	assert.EqualValues(t, 1037959168, event["memory"].(common.MapStr)["heap_usage"].(map[string]interface{})["max"])
	assert.EqualValues(t, 227420472, event["memory"].(common.MapStr)["heap_usage"].(map[string]interface{})["used"])
	assert.EqualValues(t, 2555904, event["memory"].(common.MapStr)["non_heap_usage"].(map[string]interface{})["init"])
	assert.EqualValues(t, 53477376, event["memory"].(common.MapStr)["non_heap_usage"].(map[string]interface{})["committed"])
	assert.EqualValues(t, -1, event["memory"].(common.MapStr)["non_heap_usage"].(map[string]interface{})["max"])
	assert.EqualValues(t, 50519768, event["memory"].(common.MapStr)["non_heap_usage"].(map[string]interface{})["used"])
}
