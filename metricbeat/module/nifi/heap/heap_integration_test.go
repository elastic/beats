package heap

import (
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/metricbeat/module/nifi"
)

func TestFetch(t *testing.T) {
	f := mbtest.NewEventFetcher(t, getConfig())
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

	// Check event fields are present and correct
	totalNonHeap := event["total_non_heap"].(string)
	assert.NotEqual(t, totalNonHeap, "")

	totalNonHeapBytes := event["total_non_heap_bytes"].(int64)
	assert.True(t, totalNonHeapBytes > 0)

	usedNonHeap := event["used_non_heap"].(string)
	assert.NotEqual(t, usedNonHeap, "")

	usedNonHeapBytes := event["used_non_heap_bytes"].(int64)
	assert.True(t, usedNonHeapBytes > 0)

	freeNonHeap := event["used_non_heap"].(string)
	assert.NotEqual(t, freeNonHeap, "")

	freeNonHeapBytes := event["used_non_heap_bytes"].(int64)
	assert.True(t, freeNonHeapBytes > 0)

	maxNonHeap := event["max_non_heap"].(string)
	assert.NotEqual(t, maxNonHeap, "")

	// this value may be negative, so just verify the field exists in the event
	if _, ok := event["max_non_heap_bytes"]; !ok {
		assert.Fail(t, "field [max_non_heap_bytes] no present in response")
	}

	totalHeap := event["total_heap"].(string)
	assert.NotEqual(t, totalHeap, "")

	totalHeapBytes := event["total_heap_bytes"].(int64)
	assert.True(t, totalHeapBytes > 0)

	usedHeap := event["used_heap"].(string)
	assert.NotEqual(t, usedHeap, "")

	usedHeapBytes := event["used_heap_bytes"].(int64)
	assert.True(t, usedHeapBytes > 0)

	freeHeap := event["free_heap"].(string)
	assert.NotEqual(t, freeHeap, "")

	freeHeapBytes := event["free_heap_bytes"].(int64)
	assert.True(t, freeHeapBytes > 0)

	maxHeap := event["max_heap"].(string)
	assert.NotEqual(t, maxHeap, "")

	maxHeapBytes := event["max_heap_bytes"].(int64)
	assert.True(t, maxHeapBytes > 0)

	heapUtilization := event["heap_utilization"].(string)
	assert.NotEqual(t, heapUtilization, "")
}

func TestData(t *testing.T) {
	f := mbtest.NewEventFetcher(t, getConfig())
	err := mbtest.WriteEvent(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "nifi",
		"metricsets": []string{"heap"},
		"hosts":      []string{nifi.GetEnvHost() + ":" + nifi.GetEnvPort()},
	}
}
