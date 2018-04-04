// +build !integration

package pending_tasks

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
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
	file := "./_meta/test/tasks.json"
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
	file := "./_meta/test/tasks.json"
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
			"pending_tasks_total":           0,
			"pending_tasks_priority_urgent": 0,
			"pending_tasks_priority_high":   0,
			"max_time_in_queue_millis":      0.,
		}},
		{"./_meta/test/task.json", common.MapStr{
			"pending_tasks_total":           1,
			"pending_tasks_priority_urgent": 1,
			"pending_tasks_priority_high":   0,
			"max_time_in_queue_millis":      86.,
		}},
		{
			"./_meta/test/tasks.json", common.MapStr{
				"pending_tasks_total":           3,
				"pending_tasks_priority_urgent": 1,
				"pending_tasks_priority_high":   2,
				"max_time_in_queue_millis":      858.,
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

//Events Mapping

func TestEmptyQueueShouldGiveNoError(t *testing.T) {
	file := "./_meta/test/empty.json"
	content, err := ioutil.ReadFile(file)
	assert.NoError(t, err)

	_, errs := eventsMapping(content)

	errors, ok := errs.(*s.Errors)
	if ok {
		assert.False(t, errors.HasRequiredErrors(), "mapping error: %s", errors)
	} else {
		t.Error(err)
	}
}

func TestNotEmptyQueueShouldGiveNoError(t *testing.T) {
	file := "./_meta/test/tasks.json"
	content, err := ioutil.ReadFile(file)
	assert.NoError(t, err)

	_, errs := eventsMapping(content)

	errors, ok := errs.(*s.Errors)
	if ok {
		assert.False(t, errors.HasRequiredErrors(), "mapping error: %s", errors)
	} else {
		t.Error(err)
	}
}

func TestEmptyQueueShouldGiveZeroEvent(t *testing.T) {
	file := "./_meta/test/empty.json"
	content, err := ioutil.ReadFile(file)
	assert.NoError(t, err)

	events, _ := eventsMapping(content)

	assert.Zero(t, len(events))
}

func TestNotEmptyQueueShouldGiveSeveralEvents(t *testing.T) {
	file := "./_meta/test/tasks.json"
	content, err := ioutil.ReadFile(file)
	assert.NoError(t, err)

	events, _ := eventsMapping(content)

	assert.Equal(t, 3, len(events))
}

func TestInvalidJsonForRequiredFieldShouldThrowError(t *testing.T) {
	file := "./_meta/test/invalid_required_field.json"
	content, err := ioutil.ReadFile(file)
	assert.NoError(t, err)

	_, errs := eventsMapping(content)

	errors, ok := errs.(*s.Errors)
	if ok {
		assert.True(t, errors.HasRequiredErrors(), "mapping error: %s", errors)
		assert.EqualError(t, errors, "Required fields are missing: ,source")
	} else {
		t.Error(err)
	}
}

func TestInvalidJsonForBadFormatShouldThrowError(t *testing.T) {
	file := "./_meta/test/invalid_format.json"
	content, err := ioutil.ReadFile(file)
	assert.NoError(t, err)

	_, err = eventsMapping(content)

	assert.Error(t, err)
}

func TestEventsMappedMatchToContentReceived(t *testing.T) {
	testCases := []struct {
		given    string
		expected []common.MapStr
	}{
		{"./_meta/test/empty.json", []common.MapStr(nil)},
		{"./_meta/test/task.json", []common.MapStr{common.MapStr{
			"priority":             "URGENT",
			"source":               "create-index [foo_9], cause [api]",
			"time_in_queue_millis": int64(86),
			"time_in_queue":        "86ms",
			"insert_order":         int64(101),
		}}},
		{"./_meta/test/tasks.json", []common.MapStr{common.MapStr{
			"priority":             "URGENT",
			"source":               "create-index [foo_9], cause [api]",
			"time_in_queue_millis": int64(86),
			"time_in_queue":        "86ms",
			"insert_order":         int64(101)},
			common.MapStr{
				"priority":             "HIGH",
				"source":               "shard-started ([foo_2][1], node[tMTocMvQQgGCkj7QDHl3OA], [P], s[INITIALIZING]), reason [after recovery from shard_store]",
				"time_in_queue_millis": int64(842),
				"time_in_queue":        "842ms",
				"insert_order":         int64(46),
			}, common.MapStr{
				"priority":             "HIGH",
				"source":               "shard-started ([foo_2][0], node[tMTocMvQQgGCkj7QDHl3OA], [P], s[INITIALIZING]), reason [after recovery from shard_store]",
				"time_in_queue_millis": int64(858),
				"time_in_queue":        "858ms",
				"insert_order":         int64(45),
			}}},
	}

	for _, testCase := range testCases {
		content, err := ioutil.ReadFile(testCase.given)
		assert.NoError(t, err)

		events, _ := eventsMapping(content)

		if !reflect.DeepEqual(testCase.expected, events) {
			t.Errorf("Expected %v, actual: %v", testCase.expected, events)
		}
	}
}
