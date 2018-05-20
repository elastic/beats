// +build !integration

package pending_tasks

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
)

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
	file := "./_meta/test/tasks.622.json"
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

func TestEmptyQueueShouldGiveNilEvent(t *testing.T) {
	file := "./_meta/test/empty.json"
	content, err := ioutil.ReadFile(file)
	assert.NoError(t, err)

	events, _ := eventsMapping(content)

	assert.Nil(t, events)
}

func TestNotEmptyQueueShouldGiveSeveralEvents(t *testing.T) {
	file := "./_meta/test/tasks.622.json"
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
		{"./_meta/test/task.622.json", []common.MapStr{common.MapStr{
			"priority":         "URGENT",
			"source":           "create-index [foo_9], cause [api]",
			"time_in_queue.ms": int64(86),
			"insert_order":     int64(101),
		}}},
		{"./_meta/test/tasks.622.json", []common.MapStr{common.MapStr{
			"priority":         "URGENT",
			"source":           "create-index [foo_9], cause [api]",
			"time_in_queue.ms": int64(86),
			"insert_order":     int64(101)},
			common.MapStr{
				"priority":         "HIGH",
				"source":           "shard-started ([foo_2][1], node[tMTocMvQQgGCkj7QDHl3OA], [P], s[INITIALIZING]), reason [after recovery from shard_store]",
				"time_in_queue.ms": int64(842),
				"insert_order":     int64(46),
			}, common.MapStr{
				"priority":         "HIGH",
				"source":           "shard-started ([foo_2][0], node[tMTocMvQQgGCkj7QDHl3OA], [P], s[INITIALIZING]), reason [after recovery from shard_store]",
				"time_in_queue.ms": int64(858),
				"insert_order":     int64(45),
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
