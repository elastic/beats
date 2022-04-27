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

//go:build integration
// +build integration

package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

var jsonExample = `{
    "Timestamp": "2019-02-01 11:08:50 +0000 UTC",
    "Gauges": [
        {
            "Name": "consul.autopilot.failure_tolerance",
            "Value": 0,
            "Labels": {}
        },
        {
            "Name": "consul.autopilot.healthy",
            "Value": 1,
            "Labels": {}
        },
        {
            "Name": "consul.runtime.alloc_bytes",
            "Value": 5034304,
            "Labels": {}
        },
        {
            "Name": "consul.runtime.free_count",
            "Value": 1202914,
            "Labels": {}
        },
        {
            "Name": "consul.runtime.heap_objects",
            "Value": 35836,
            "Labels": {"service":"service1"}
        },
        {
            "Name": "consul.runtime.malloc_count",
            "Value": 1238750,
            "Labels": {}
        },
        {
            "Name": "consul.runtime.num_goroutines",
            "Value": 76,
            "Labels": {}
        },
        {
            "Name": "consul.runtime.sys_bytes",
            "Value": 73070840,
            "Labels": {}
        },
        {
            "Name": "consul.runtime.total_gc_pause_ns",
            "Value": 7107735,
            "Labels": {}
        },
        {
            "Name": "consul.runtime.total_gc_runs",
            "Value": 42,
            "Labels": {}
        },
        {
            "Name": "consul.session_ttl.active",
            "Value": 0,
            "Labels": {}
        }
    ],
    "Points": [],
    "Counters": [
        {
            "Name": "consul.raft.apply",
            "Count": 1,
            "Rate": 0.1,
            "Sum": 1,
            "Min": 1,
            "Max": 1,
            "Mean": 1,
            "Stddev": 0,
            "Labels": {}
        },
        {
            "Name": "consul.rpc.query",
            "Count": 2,
            "Rate": 0.2,
            "Sum": 2,
            "Min": 1,
            "Max": 1,
            "Mean": 1,
            "Stddev": 0,
            "Labels": {}
        },
        {
            "Name": "consul.rpc.request",
            "Count": 5,
            "Rate": 0.5,
            "Sum": 5,
            "Min": 1,
            "Max": 1,
            "Mean": 1,
            "Stddev": 0,
            "Labels": {}
        }
    ],
    "Samples": [
        {
            "Name": "consul.fsm.coordinate.batch-update",
            "Count": 1,
            "Rate": 0.003936899825930595,
            "Sum": 0.039368998259305954,
            "Min": 0.039368998259305954,
            "Max": 0.039368998259305954,
            "Mean": 0.039368998259305954,
            "Stddev": 0,
            "Labels": {}
        },
        {
            "Name": "consul.http.GET.v1.agent.metrics",
            "Count": 10,
            "Rate": 0.2068565994501114,
            "Sum": 2.068565994501114,
            "Min": 0.14361299574375153,
            "Max": 0.46759501099586487,
            "Mean": 0.2068565994501114,
            "Stddev": 0.09421784218829098,
            "Labels": {}
        },
        {
            "Name": "consul.memberlist.gossip",
            "Count": 200,
            "Rate": 0.2729187995195389,
            "Sum": 2.729187995195389,
            "Min": 0.0022559999488294125,
            "Max": 0.10744299739599228,
            "Mean": 0.013645939975976944,
            "Stddev": 0.013672823772901079,
            "Labels": {}
        },
        {
            "Name": "consul.raft.commitTime",
            "Count": 1,
            "Rate": 0.00219310000538826,
            "Sum": 0.0219310000538826,
            "Min": 0.0219310000538826,
            "Max": 0.0219310000538826,
            "Mean": 0.0219310000538826,
            "Stddev": 0,
            "Labels": {}
        },
        {
            "Name": "consul.raft.fsm.apply",
            "Count": 1,
            "Rate": 0.005605699867010117,
            "Sum": 0.056056998670101166,
            "Min": 0.056056998670101166,
            "Max": 0.056056998670101166,
            "Mean": 0.056056998670101166,
            "Stddev": 0,
            "Labels": {}
        },
        {
            "Name": "consul.raft.leader.dispatchLog",
            "Count": 1,
            "Rate": 0.001807899959385395,
            "Sum": 0.01807899959385395,
            "Min": 0.01807899959385395,
            "Max": 0.01807899959385395,
            "Mean": 0.01807899959385395,
            "Stddev": 0,
            "Labels": {}
        },
        {
            "Name": "consul.runtime.gc_pause_ns",
            "Count": 1,
            "Rate": 12140.1,
            "Sum": 121401,
            "Min": 121401,
            "Max": 121401,
            "Mean": 121401,
            "Stddev": 0,
            "Labels": {}
        },
        {
            "Name": "consul.serf.queue.Event",
            "Count": 2,
            "Rate": 0.1,
            "Sum": 1,
            "Min": 0,
            "Max": 1,
            "Mean": 0.5,
            "Stddev": 0.7071067811865476,
            "Labels": {}
        },
        {
            "Name": "consul.serf.queue.Intent",
            "Count": 2,
            "Rate": 0,
            "Sum": 0,
            "Min": 0,
            "Max": 0,
            "Mean": 0,
            "Stddev": 0,
            "Labels": {}
        },
        {
            "Name": "consul.serf.queue.Query",
            "Count": 2,
            "Rate": 0,
            "Sum": 0,
            "Min": 0,
            "Max": 0,
            "Mean": 0,
            "Stddev": 0,
            "Labels": {}
        }
    ]
}`

func TestEventMapping(t *testing.T) {
	byt := []byte(jsonExample)

	events, err := eventMapping(byt)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(events))

	//2 events should be here, one with runtime.heap_objects only and one with everything else
	heapObjectsFound := false
	var heapObjects interface{}

	goroutinesFound := false
	var goroutines interface{}

	for _, event := range events {
		runtimeI, ok := event["runtime"]
		assert.True(t, ok)

		runtime, ok := runtimeI.(mapstr.M)
		assert.True(t, ok)

		//do not overwrite if heapObjectsFound has already been set to true
		if !heapObjectsFound {
			heapObjects, heapObjectsFound = runtime["heap_objects"]
			if heapObjectsFound {
				heapObjectsFloat64, ok := heapObjects.(float64)
				assert.True(t, ok)

				assert.True(t, heapObjectsFloat64 > 0)
			}
		}

		//do not overwrite if goroutinesFound has already been set to true
		if !goroutinesFound {
			goroutines, goroutinesFound = runtime["goroutines"]
			if goroutinesFound {
				goroutinesFloat64, ok := goroutines.(float64)
				assert.True(t, ok)

				assert.True(t, goroutinesFloat64 > 0)
			}
		}
	}

	assert.True(t, goroutinesFound)
	assert.True(t, heapObjectsFound)
}

func TestUniqueKeyForLabelMap(t *testing.T) {
	input := []map[string]string{
		{
			"a": "b",
			"g": "h",
			"c": "d",
			"e": "f",
			"i": "j",
		},
		{
			"a": "b",
			"e": "f",
			"c": "d",
			"g": "h",
			"i": "j",
		},
		{
			"c": "d",
			"a": "b",
			"g": "h",
			"e": "f",
			"i": "j",
		},
		{
			"c": "d",
			"e": "f",
			"i": "j",
			"a": "b",
			"g": "h",
		},
		{
			"e": "f",
			"a": "b",
			"c": "d",
			"g": "h",
			"i": "j",
		},
		{
			"e": "f",
			"i": "j",
			"c": "d",
			"a": "b",
			"g": "h",
		},
	}

	keys := make([]string, 0)
	for _, i := range input {
		keys = append(keys, uniqueKeyForLabelMap(i))
	}

	for i := 1; i < len(keys); i++ {
		assert.True(t, keys[i-1] == keys[i])
	}
}
