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

package consumergroup

import (
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/common"
)

func TestFetchGroupInfo(t *testing.T) {
	noEvents := func(events []common.MapStr) {
		assert.Len(t, events, 0)
	}

	tests := []struct {
		name     string
		client   client
		groups   []string
		topics   []string
		err      error
		expected []common.MapStr
		validate func([]common.MapStr)
	}{
		{
			name: "Test all groups",
			client: defaultMockClient(mockState{
				partitions: map[string]map[string][]int64{
					"group1": {
						"topic1": {10, 11, 12},
						"topic3": {6, 7},
					},
					"group2": {
						"topic2": {3},
						"topic3": {9, 10},
					},
				},
				groups: map[string][]map[string][]int32{
					"group1": {
						{"topic1": {0, 2}, "topic3": {1}},
						{"topic1": {1}, "topic3": {0}},
					},
					"group2": {
						{"topic2": {0}, "topic3": {0, 1}},
					},
				},
			}),
			expected: []common.MapStr{
				testEvent("group1", "topic1", 0, common.MapStr{
					"client":       clientMeta(0),
					"offset":       int64(10),
					"consumer_lag": int64(42) - int64(10),
				}),
				testEvent("group1", "topic1", 1, common.MapStr{
					"client":       clientMeta(1),
					"offset":       int64(11),
					"consumer_lag": int64(42) - int64(11),
				}),
				testEvent("group1", "topic1", 2, common.MapStr{
					"client":       clientMeta(0),
					"offset":       int64(12),
					"consumer_lag": int64(42) - int64(12),
				}),
				testEvent("group1", "topic3", 0, common.MapStr{
					"client":       clientMeta(1),
					"offset":       int64(6),
					"consumer_lag": int64(42) - int64(6),
				}),
				testEvent("group1", "topic3", 1, common.MapStr{
					"client":       clientMeta(0),
					"offset":       int64(7),
					"consumer_lag": int64(42) - int64(7),
				}),
				testEvent("group2", "topic2", 0, common.MapStr{
					"client":       clientMeta(0),
					"offset":       int64(3),
					"consumer_lag": int64(42) - int64(3),
				}),
				testEvent("group2", "topic3", 0, common.MapStr{
					"client":       clientMeta(0),
					"offset":       int64(9),
					"consumer_lag": int64(42) - int64(9),
				}),
				testEvent("group2", "topic3", 1, common.MapStr{
					"client":       clientMeta(0),
					"offset":       int64(10),
					"consumer_lag": int64(42) - int64(10),
				}),
			},
		},

		{
			name: "filter topics and groups",
			client: defaultMockClient(mockState{
				partitions: map[string]map[string][]int64{
					"group1": {
						"topic1": {1, 2},
						"topic2": {3, 4},
					},
					"group2": {
						"topic2": {5, 6},
						"topic3": {7, 8},
					},
				},
				groups: map[string][]map[string][]int32{
					"group1": {
						{"topic1": {0, 1}, "topic2": {0, 1}},
					},
					"group2": {
						{"topic1": {0, 1}, "topic2": {0, 1}},
					},
				},
			}),
			groups: []string{"group1"},
			topics: []string{"topic1"},
			expected: []common.MapStr{
				testEvent("group1", "topic1", 0, common.MapStr{
					"client":       clientMeta(0),
					"offset":       int64(1),
					"consumer_lag": int64(42) - int64(1),
				}),
				testEvent("group1", "topic1", 1, common.MapStr{
					"client":       clientMeta(0),
					"offset":       int64(2),
					"consumer_lag": int64(42) - int64(2),
				}),
			},
		},

		{
			name:     "no events on empty group",
			client:   defaultMockClient(mockState{}),
			validate: noEvents,
		},

		{
			name: "fail to list groups",
			client: defaultMockClient(mockState{}).with(func(c *mockClient) {
				c.listGroups = func() ([]string, error) {
					return nil, io.EOF
				}
			}),
			err:      io.EOF,
			validate: noEvents,
		},

		{
			name: "fail if assignment query fails",
			client: defaultMockClient(mockState{
				partitions: map[string]map[string][]int64{
					"group1": {"topic1": {1}},
				},
				groups: map[string][]map[string][]int32{
					"group1": {{"topic1": {0}}},
				},
			}).with(func(c *mockClient) {
				c.describeGroups = makeDescribeGroupsFail(io.EOF)
			}),
			err:      io.EOF,
			validate: noEvents,
		},

		{
			name: "fail when fetching group offsets",
			client: defaultMockClient(mockState{
				partitions: map[string]map[string][]int64{
					"group1": {"topic1": {1}},
				},
				groups: map[string][]map[string][]int32{
					"group1": {{"topic1": {0}}},
				},
			}).with(func(c *mockClient) {
				c.fetchGroupOffsets = makeFetchGroupOffsetsFail(io.EOF)
			}),
			err:      io.EOF,
			validate: noEvents,
		},
	}

	for i, test := range tests {
		t.Logf("run test (%v): %v", i, test.name)

		var events []common.MapStr
		collectEvents := func(event common.MapStr) {
			t.Logf("new event: %v", event)
			events = append(events, event)
		}

		indexEvents := func(events []common.MapStr) map[string]common.MapStr {
			index := map[string]common.MapStr{}
			for _, e := range events {
				key := fmt.Sprintf("%v::%v::%v",
					e["id"], e["topic"], e["partition"],
				)
				index[key] = e
			}
			return index
		}

		groups := makeNameSet(test.groups...).pred()
		topics := makeNameSet(test.topics...).pred()
		err := fetchGroupInfo(collectEvents, test.client, groups, topics)
		if err != nil {
			switch {
			case test.err == nil:
				t.Fatal(err)
			case test.err != err:
				t.Error(err)
			}
			continue
		}

		indexed := indexEvents(events)
		for key, expected := range indexEvents(test.expected) {
			event, found := indexed[key]
			if !found {
				t.Errorf("Missing key %v from events: %v", key, events)
				continue
			}
			assertEvent(t, expected, event)
		}

		if test.validate != nil {
			test.validate(events)
		}
	}
}

func assertEvent(t *testing.T, expected, event common.MapStr) {
	for field, exp := range expected {
		val, found := event[field]
		if !found {
			t.Errorf("Missing field: %v", field)
			continue
		}

		if sub, ok := exp.(common.MapStr); ok {
			assertEvent(t, sub, val.(common.MapStr))
		} else {
			if !assert.Equal(t, exp, val) {
				t.Logf("failed in field: %v", field)
				t.Logf("type expected: %v", reflect.TypeOf(exp))
				t.Logf("type event: %v", reflect.TypeOf(val))
				t.Logf("------------------------------")
			}
		}
	}
}

func testEvent(
	group, topic string,
	partition int,
	fields ...common.MapStr,
) common.MapStr {
	event := common.MapStr{
		"id":        group,
		"topic":     topic,
		"partition": int32(partition),
	}

	for _, extra := range fields {
		for k, v := range extra {
			event[k] = v
		}
	}
	return event
}

func clientMeta(id int) common.MapStr {
	return common.MapStr{
		"id": fmt.Sprintf("consumer-%v", id),
	}
}
