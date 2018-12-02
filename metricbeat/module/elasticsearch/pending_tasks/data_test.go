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

// +build !integration

package pending_tasks

import (
	"io/ioutil"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

var info = elasticsearch.Info{
	ClusterID:   "1234",
	ClusterName: "helloworld",
}

//Events Mapping

func TestEmptyQueueShouldGiveNoError(t *testing.T) {
	file := "./_meta/test/empty.json"
	content, err := ioutil.ReadFile(file)
	assert.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	err = eventsMapping(reporter, info, content)
	assert.NoError(t, err)
}

func TestNotEmptyQueueShouldGiveNoError(t *testing.T) {
	file := "./_meta/test/tasks.622.json"
	content, err := ioutil.ReadFile(file)
	assert.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	err = eventsMapping(reporter, info, content)
	assert.NoError(t, err)
	assert.True(t, len(reporter.GetEvents()) >= 1)
	assert.Zero(t, len(reporter.GetErrors()))
}

func TestEmptyQueueShouldGiveZeroEvent(t *testing.T) {
	file := "./_meta/test/empty.json"
	content, err := ioutil.ReadFile(file)
	assert.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	err = eventsMapping(reporter, info, content)
	assert.Zero(t, len(reporter.GetEvents()))
	assert.Zero(t, len(reporter.GetErrors()))
}

func TestNotEmptyQueueShouldGiveSeveralEvents(t *testing.T) {
	file := "./_meta/test/tasks.622.json"
	content, err := ioutil.ReadFile(file)
	assert.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	err = eventsMapping(reporter, info, content)
	assert.Equal(t, 3, len(reporter.GetEvents()))
	assert.Zero(t, len(reporter.GetErrors()))
}

func TestInvalidJsonForRequiredFieldShouldThrowError(t *testing.T) {
	file := "./_meta/test/invalid_required_field.json"
	content, err := ioutil.ReadFile(file)
	assert.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	err = eventsMapping(reporter, info, content)
	assert.Error(t, err)
}

func TestInvalidJsonForBadFormatShouldThrowError(t *testing.T) {
	file := "./_meta/test/invalid_format.json"
	content, err := ioutil.ReadFile(file)
	assert.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	err = eventsMapping(reporter, info, content)
	assert.Error(t, err)
}

func TestEventsMappedMatchToContentReceived(t *testing.T) {
	testCases := []struct {
		given    string
		expected []mb.Event
	}{
		{"./_meta/test/empty.json", []mb.Event(nil)},
		{"./_meta/test/task.622.json", []mb.Event{
			mb.Event{
				RootFields: common.MapStr{
					"service": common.MapStr{
						"name": "elasticsearch",
					},
				},
				ModuleFields: common.MapStr{
					"cluster": common.MapStr{
						"id":   "1234",
						"name": "helloworld",
					},
				},
				MetricSetFields: common.MapStr{
					"priority":         "URGENT",
					"source":           "create-index [foo_9], cause [api]",
					"time_in_queue.ms": int64(86),
					"insert_order":     int64(101),
				},
				Timestamp: time.Time{},
				Took:      0,
			},
		}},
		{"./_meta/test/tasks.622.json", []mb.Event{
			mb.Event{
				RootFields: common.MapStr{
					"service": common.MapStr{
						"name": "elasticsearch",
					},
				},
				ModuleFields: common.MapStr{
					"cluster": common.MapStr{
						"id":   "1234",
						"name": "helloworld",
					},
				},
				MetricSetFields: common.MapStr{
					"priority":         "URGENT",
					"source":           "create-index [foo_9], cause [api]",
					"time_in_queue.ms": int64(86),
					"insert_order":     int64(101),
				},
				Timestamp: time.Time{},
				Took:      0,
			},
			mb.Event{
				RootFields: common.MapStr{
					"service": common.MapStr{
						"name": "elasticsearch",
					},
				},
				ModuleFields: common.MapStr{
					"cluster": common.MapStr{
						"id":   "1234",
						"name": "helloworld",
					},
				},
				MetricSetFields: common.MapStr{"priority": "HIGH",
					"source":           "shard-started ([foo_2][1], node[tMTocMvQQgGCkj7QDHl3OA], [P], s[INITIALIZING]), reason [after recovery from shard_store]",
					"time_in_queue.ms": int64(842),
					"insert_order":     int64(46),
				},
				Timestamp: time.Time{},
				Took:      0,
			}, mb.Event{
				RootFields: common.MapStr{
					"service": common.MapStr{
						"name": "elasticsearch",
					},
				},
				ModuleFields: common.MapStr{
					"cluster": common.MapStr{
						"id":   "1234",
						"name": "helloworld",
					},
				},
				MetricSetFields: common.MapStr{
					"priority":         "HIGH",
					"source":           "shard-started ([foo_2][0], node[tMTocMvQQgGCkj7QDHl3OA], [P], s[INITIALIZING]), reason [after recovery from shard_store]",
					"time_in_queue.ms": int64(858),
					"insert_order":     int64(45),
				}, Timestamp: time.Time{},
				Took: 0,
			},
		}},
	}

	for _, testCase := range testCases {
		content, err := ioutil.ReadFile(testCase.given)
		assert.NoError(t, err)

		reporter := &mbtest.CapturingReporterV2{}
		err = eventsMapping(reporter, info, content)

		events := reporter.GetEvents()
		if !reflect.DeepEqual(testCase.expected, events) {
			t.Errorf("Expected %v, actual: %v", testCase.expected, events)
		}
	}
}
