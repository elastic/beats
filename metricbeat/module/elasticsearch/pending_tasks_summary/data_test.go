// +build !integration

package pending_tasks_summary

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

// Event Mapping

func TestEmptyQueueShouldGiveNoErrorMappedEvent(t *testing.T) {
	file := "./_meta/test/empty.json"
	content, err := ioutil.ReadFile(file)
	assert.NoError(t, err)

	_, err = eventMapping(content)

	assert.NoError(t, err)
}

func TestNotEmptyQueueShouldGiveNoErrorWithMappedEvent(t *testing.T) {
	file := "./_meta/test/tasks.622.json"
	content, err := ioutil.ReadFile(file)
	assert.NoError(t, err)

	_, err = eventMapping(content)

	assert.NoError(t, err)
}

func TestEmptyQueueShouldGiveAnEventWithMappedEvent(t *testing.T) {
	file := "./_meta/test/empty.json"
	content, err := ioutil.ReadFile(file)
	assert.NoError(t, err)

	event, _ := eventMapping(content)

	assert.NotNil(t, event)
}

func TestNotEmptyQueueWithSeveralTasksShouldGiveOneEventWithMappedEvent(t *testing.T) {
	file := "./_meta/test/tasks.622.json"
	content, err := ioutil.ReadFile(file)
	assert.NoError(t, err)

	event, _ := eventMapping(content)

	assert.NotNil(t, event)
}

func TestEventMappedMatchToContentReceived(t *testing.T) {
	testCases := []struct {
		given    string
		expected common.MapStr
	}{
		{"./_meta/test/empty.json", common.MapStr{
			"count_total":              0,
			"count_priority_urgent":    0,
			"count_priority_high":      0,
			"max_time_in_queue_millis": 0.,
		}},
		{"./_meta/test/task.622.json", common.MapStr{
			"count_total":              1,
			"count_priority_urgent":    1,
			"count_priority_high":      0,
			"max_time_in_queue_millis": 86.,
		}},
		{
			"./_meta/test/tasks.622.json", common.MapStr{
				"count_total":              3,
				"count_priority_urgent":    1,
				"count_priority_high":      2,
				"max_time_in_queue_millis": 858.,
			}},
	}

	for _, testCase := range testCases {
		content, err := ioutil.ReadFile(testCase.given)
		assert.NoError(t, err)

		event, _ := eventMapping(content)

		if !reflect.DeepEqual(testCase.expected, event) {
			t.Errorf("Expected %+v, actual: %+v", testCase.expected, event)
		}
	}
}
