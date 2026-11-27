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

//go:build windows

package perfmon

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/metricbeat/helper/windows/pdh"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/mapstr"
)

func TestGroupErrors(t *testing.T) {
	reader := Reader{
		config: Config{
			GroupMeasurements: true,
		},
		query: pdh.Query{},
		log:   nil,
		counters: []PerfCounter{
			{
				QueryField:   "datagrams_sent_per_sec",
				QueryName:    `\UDPv4\Datagrams Sent/sec`,
				Format:       "float",
				ObjectName:   "UDPv4",
				ObjectField:  "object",
				ChildQueries: []string{`\UDPv4\Datagrams Sent/sec`},
			},
			{
				QueryField:    "%_processor_time",
				QueryName:     `\Processor Information(_Total)\% Processor Time`,
				Format:        "float",
				ObjectName:    "Processor Information",
				ObjectField:   "object",
				InstanceName:  "_Total",
				InstanceField: "instance",
				ChildQueries:  []string{`\Processor Information(_Total)\% Processor Time`},
			},
			{
				QueryField:    "current_disk_queue_length",
				QueryName:     `\PhysicalDisk(_Total)\Current Disk Queue Length`,
				Format:        "float",
				ObjectName:    "PhysicalDisk",
				ObjectField:   "object",
				InstanceName:  "_Total",
				InstanceField: "instance",
				ChildQueries:  []string{`\PhysicalDisk(_Total)\Current Disk Queue Length`},
			},
		},
	}

	counters := map[string][]pdh.CounterValue{
		`\UDPv4\Datagrams Sent/sec`: {
			{Instance: "", Measurement: 23},
		},
		`\Processor Information(_Total)\% Processor Time`: {
			{Instance: "_Total", Measurement: 11},
		},
		`\PhysicalDisk(_Total)\Current Disk Queue Length`: {
			{Instance: "_Total", Measurement: 20},
		},
	}

	events := reader.groupToEvents(counters)
	assert.NotNil(t, events)
	assert.Equal(t, 3, len(events))

	for _, event := range events {
		assert.NoError(t, event.Error)
	}

}

func TestGroupToEvents(t *testing.T) {
	reader := Reader{
		config: Config{
			GroupMeasurements: true,
		},
		query: pdh.Query{},
		log:   nil,
		counters: []PerfCounter{
			{
				QueryField:   "datagrams_sent_per_sec",
				QueryName:    `\UDPv4\Datagrams Sent/sec`,
				Format:       "float",
				ObjectName:   "UDPv4",
				ObjectField:  "object",
				ChildQueries: []string{`\UDPv4\Datagrams Sent/sec`},
			},
			{
				QueryField:    "%_processor_time",
				QueryName:     `\Processor Information(_Total)\% Processor Time`,
				Format:        "float",
				ObjectName:    "Processor Information",
				ObjectField:   "object",
				InstanceName:  "_Total",
				InstanceField: "instance",
				ChildQueries:  []string{`\Processor Information(_Total)\% Processor Time`},
			},
			{
				QueryField:    "current_disk_queue_length",
				QueryName:     `\PhysicalDisk(_Total)\Current Disk Queue Length`,
				Format:        "float",
				ObjectName:    "PhysicalDisk",
				ObjectField:   "object",
				InstanceName:  "_Total",
				InstanceField: "instance",
				ChildQueries:  []string{`\PhysicalDisk(_Total)\Current Disk Queue Length`},
			},
		},
	}
	counters := map[string][]pdh.CounterValue{
		`\UDPv4\Datagrams Sent/sec`: {
			{
				Instance:    "",
				Measurement: 23,
				Err:         pdh.CounterValueError{},
			},
		},
		`\Processor Information(_Total)\% Processor Time`: {
			{
				Instance:    "_Total",
				Measurement: 11,
			},
		},
		`\PhysicalDisk(_Total)\Current Disk Queue Length`: {
			{
				Instance:    "_Total",
				Measurement: 20,
			},
		},
	}

	events := reader.groupToEvents(counters)
	assert.NotNil(t, events)
	assert.Equal(t, 3, len(events))

	for _, event := range events {
		ok, err := event.MetricSetFields.HasKey("datagrams_sent_per_sec")
		if ok {
			assert.NoError(t, err)
			assert.True(t, ok)
			ok, err = event.MetricSetFields.HasKey("object")
			assert.NoError(t, err)
			assert.True(t, ok)

			val, err := event.MetricSetFields.GetValue("datagrams_sent_per_sec")
			assert.NoError(t, err)
			assert.Equal(t, val, 23)

			val, err = event.MetricSetFields.GetValue("object")
			assert.NoError(t, err)
			assert.Equal(t, val, "UDPv4")
		} else {
			ok, err := event.MetricSetFields.HasKey("%_processor_time")
			if ok {
				assert.NoError(t, err)
				assert.True(t, ok)

				ok, err = event.MetricSetFields.HasKey("object")
				assert.NoError(t, err)
				assert.True(t, ok)

				val, err := event.MetricSetFields.GetValue("%_processor_time")
				assert.NoError(t, err)
				assert.Equal(t, val, 11)

				val, err = event.MetricSetFields.GetValue("object")
				assert.NoError(t, err)
				assert.Equal(t, val, "Processor Information")
			} else {
				ok, err = event.MetricSetFields.HasKey("current_disk_queue_length")
				assert.NoError(t, err)
				assert.True(t, ok)

				ok, err = event.MetricSetFields.HasKey("object")
				assert.NoError(t, err)
				assert.True(t, ok)

				val, err := event.MetricSetFields.GetValue("current_disk_queue_length")
				assert.NoError(t, err)
				assert.Equal(t, val, 20)

				val, err = event.MetricSetFields.GetValue("object")
				assert.NoError(t, err)
				assert.Equal(t, val, "PhysicalDisk")
			}
		}
	}
}

func TestGroupToSingleEvent(t *testing.T) {
	reader := Reader{
		query: pdh.Query{},
		log:   nil,
		config: Config{
			GroupAllCountersTo: "processor_count",
		},
		counters: []PerfCounter{
			{
				QueryField:    "%_processor_time",
				QueryName:     `\Processor Information(*)\% Processor Time`,
				Format:        "float",
				ObjectName:    "Processor Information",
				ObjectField:   "object",
				InstanceName:  "*",
				InstanceField: "instance",
				ChildQueries:  []string{`\Processor Information(processor0)\% Processor Time`, `\Processor Information(processor1)\% Processor Time`},
			},
			{
				QueryField:    "%_user_time",
				QueryName:     `\Processor Information(*)\% User Time`,
				Format:        "float",
				ObjectName:    "Processor Information",
				ObjectField:   "object",
				InstanceName:  "*",
				InstanceField: "instance",
				ChildQueries:  []string{`\Processor Information(processor0)\% User Time`, `\Processor Information(processor1)\% User Time`},
			},
		},
	}

	counters := map[string][]pdh.CounterValue{
		`\Processor Information(processor0)\% Processor Time`: {
			{
				Instance:    "processor0",
				Measurement: 23,
			},
		},
		`\Processor Information(processor1)\% Processor Time`: {
			{
				Instance:    "processor1",
				Measurement: 21,
			},
		},
		`\Processor Information(processor0)\% User Time`: {
			{
				Instance:    "processor0",
				Measurement: 10,
			},
		},
		`\Processor Information(processor1)\% User Time`: {
			{
				Instance:    "processor1",
				Measurement: 11,
			},
		},
	}
	event := reader.groupToSingleEvent(counters)
	assert.NotNil(t, event)
	ok, err := event.MetricSetFields.HasKey("%_processor_time")
	assert.NoError(t, err)
	assert.True(t, ok)
	ok, err = event.MetricSetFields.HasKey("%_processor_time:count")
	assert.NoError(t, err)
	assert.True(t, ok)
	val, err := event.MetricSetFields.GetValue("%_processor_time")
	assert.NoError(t, err)
	assert.Equal(t, val, float64(44))
	val, err = event.MetricSetFields.GetValue("%_processor_time:count")
	assert.NoError(t, err)
	assert.Equal(t, val, mapstr.M{"processor_count": float64(2)})
	ok, err = event.MetricSetFields.HasKey("%_user_time")
	assert.NoError(t, err)
	assert.True(t, ok)
	ok, err = event.MetricSetFields.HasKey("%_user_time:count")
	assert.NoError(t, err)
	assert.True(t, ok)
	val, err = event.MetricSetFields.GetValue("%_user_time")
	assert.NoError(t, err)
	assert.Equal(t, val, float64(21))
	val, err = event.MetricSetFields.GetValue("%_user_time:count")
	assert.NoError(t, err)
	assert.Equal(t, val, mapstr.M{"processor_count": float64(2)})
}

func TestGroupToSingleEventInstanceCount(t *testing.T) {
	process := func(counter, field string, instances ...string) PerfCounter {
		perfCounter := PerfCounter{
			QueryField:   "process." + field,
			QueryName:    `\Process(w3wp*)\` + counter,
			ObjectName:   "Process",
			InstanceName: "w3wp*",
		}
		for _, instance := range instances {
			perfCounter.ChildQueries = append(perfCounter.ChildQueries, `\Process(`+instance+`)\`+counter)
		}
		return perfCounter
	}
	value := func(instance string, measurement any) []pdh.CounterValue {
		return []pdh.CounterValue{{Instance: instance, Measurement: measurement}}
	}
	total := PerfCounter{
		QueryField:   "network.current_connections",
		QueryName:    `\Web Service(_Total)\Current Connections`,
		ObjectName:   "Web Service",
		InstanceName: "_Total",
		ChildQueries: []string{`\Web Service(_Total)\Current Connections`},
	}
	noInstance := PerfCounter{
		QueryField:   "cache.uri_cache_hits",
		QueryName:    `\Web Service Cache\URI Cache Hits`,
		ObjectName:   "Web Service Cache",
		ChildQueries: []string{`\Web Service Cache\URI Cache Hits`},
	}

	tests := map[string]struct {
		counters []PerfCounter
		values   map[string][]pdh.CounterValue
		want     mapstr.M
	}{
		"one instance of a wildcard": {
			counters: []PerfCounter{process("Handle Count", "handle_count", "w3wp")},
			values: map[string][]pdh.CounterValue{
				`\Process(w3wp)\Handle Count`: value("w3wp", 120),
			},
			want: mapstr.M{"process": mapstr.M{"handle_count": 120.0, "worker_process_count": 1.0}},
		},
		"several instances of a wildcard": {
			counters: []PerfCounter{process("Handle Count", "handle_count", "w3wp", "w3wp#1")},
			values: map[string][]pdh.CounterValue{
				`\Process(w3wp)\Handle Count`:   value("w3wp", 120),
				`\Process(w3wp#1)\Handle Count`: value("w3wp#1", 30),
			},
			want: mapstr.M{"process": mapstr.M{"handle_count": 150.0, "worker_process_count": 2.0}},
		},
		"counter without a value for every instance": {
			counters: []PerfCounter{
				process("Handle Count", "handle_count", "w3wp", "w3wp#1"),
				process("% Processor Time", "cpu_usage_perc", "w3wp", "w3wp#1"),
			},
			values: map[string][]pdh.CounterValue{
				`\Process(w3wp)\Handle Count`:     value("w3wp", 120),
				`\Process(w3wp#1)\Handle Count`:   value("w3wp#1", 30),
				`\Process(w3wp)\% Processor Time`: value("w3wp", 2.5),
				`\Process(w3wp#1)\% Processor Time`: {
					{Instance: "w3wp#1", Err: pdh.CounterValueError{Error: pdh.PDH_CALC_NEGATIVE_VALUE}},
				},
			},
			want: mapstr.M{"process": mapstr.M{"handle_count": 150.0, "cpu_usage_perc": 2.5, "worker_process_count": 2.0}},
		},
		"no instances of a wildcard": {
			counters: []PerfCounter{process("Handle Count", "handle_count"), total},
			values: map[string][]pdh.CounterValue{
				`\Web Service(_Total)\Current Connections`: value("_Total", 7),
			},
			want: mapstr.M{"network": mapstr.M{"current_connections": 7.0}},
		},
		"counters that are not a wildcard": {
			counters: []PerfCounter{total, noInstance},
			values: map[string][]pdh.CounterValue{
				`\Web Service(_Total)\Current Connections`: value("_Total", 7),
				`\Web Service Cache\URI Cache Hits`:        value("", 3),
			},
			want: mapstr.M{
				"network": mapstr.M{"current_connections": 7.0},
				"cache":   mapstr.M{"uri_cache_hits": 3.0},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			reader := Reader{
				log:      logptest.NewTestingLogger(t, ""),
				config:   Config{GroupAllCountersTo: "worker_process_count"},
				counters: test.counters,
			}
			event := reader.groupToSingleEvent(test.values)
			assert.Equal(t, test.want, event.MetricSetFields, "unexpected event fields")
		})
	}
}

func TestMatchByParentInstance(t *testing.T) {
	_true := true
	_false := false
	reader := Reader{
		query: pdh.Query{},
		log:   nil,
		config: Config{
			MatchByParentInstance: &_true,
		},
		counters: []PerfCounter{
			{
				QueryField:    "%_processor_time",
				QueryName:     `\Processor Information(*)\% Processor Time`,
				Format:        "float",
				ObjectName:    "Processor Information",
				ObjectField:   "object",
				InstanceName:  "*",
				InstanceField: "instance",
				ChildQueries:  []string{`\Processor Information(processor)\% Processor Time`, `\Processor Information(processor#1)\% Processor Time`},
			},
		},
	}

	counters := map[string][]pdh.CounterValue{
		`\Processor Information(processor)\% Processor Time`: {
			{
				Instance:    "processor",
				Measurement: 1,
			},
		},
		`\Processor Information(processor#1)\% Processor Time`: {
			{
				Instance:    "processor#1",
				Measurement: 2,
			},
		},
	}

	{
		events := reader.groupToEvents(counters)
		assert.NotNil(t, events)
		assert.Equal(t, 2, len(events))
		ok, err := events[0].MetricSetFields.HasKey("instance")
		assert.NoError(t, err)
		assert.True(t, ok)
		ok, err = events[1].MetricSetFields.HasKey("instance")
		assert.NoError(t, err)
		assert.True(t, ok)
		val1, err := events[0].MetricSetFields.GetValue("instance")
		assert.NoError(t, err)
		assert.Equal(t, val1, "processor")
		val2, err := events[1].MetricSetFields.GetValue("instance")
		assert.NoError(t, err)
		assert.Equal(t, val2, "processor")
	}

	reader.config.MatchByParentInstance = &_false
	{
		events := reader.groupToEvents(counters)
		assert.NotNil(t, events)
		assert.Equal(t, 2, len(events))
		pt, _ := events[0].MetricSetFields.GetValue("%_processor_time")
		var exp1, exp2 string
		// since counters is a map we use the value to determine order
		// of expected values
		if pt == 1 {
			exp1, exp2 = "processor", "processor#1"
		} else {
			exp1, exp2 = "processor#1", "processor"
		}
		ok, err := events[0].MetricSetFields.HasKey("instance")
		assert.NoError(t, err)
		assert.True(t, ok)
		ok, err = events[1].MetricSetFields.HasKey("instance")
		assert.NoError(t, err)
		assert.True(t, ok)
		val1, err := events[0].MetricSetFields.GetValue("instance")
		assert.NoError(t, err)
		assert.Equal(t, val1, exp1)
		val2, err := events[1].MetricSetFields.GetValue("instance")
		assert.NoError(t, err)
		assert.Equal(t, val2, exp2)
	}
}

func TestMatchesParentProcess(t *testing.T) {
	ok, val := matchesParentProcess("svchost")
	assert.True(t, ok)
	assert.Equal(t, val, "svchost")
	ok, val = matchesParentProcess("svchost#54")
	assert.True(t, ok)
	assert.Equal(t, val, "svchost")

	ok, val = matchesParentProcess("svchost (test) another #54")
	assert.True(t, ok)
	assert.Equal(t, val, "svchost (test) another #54")
}

func TestExtractObjectFromCounter(t *testing.T) {
	_true := true
	_false := false

	reader := Reader{
		query:  pdh.Query{},
		log:    nil,
		config: Config{},
		counters: []PerfCounter{
			{
				QueryField:    "working_set",
				QueryName:     `\Process(*)\% Processor Time`,
				Format:        "float",
				ObjectName:    "Process",
				ObjectField:   "object",
				InstanceName:  "*",
				InstanceField: "instance",
				ChildQueries:  []string{`\Process(chrome)\% Processor Time`},
			},
		},
	}

	counters := map[string][]pdh.CounterValue{
		`\Process(chrome)\% Processor Time`: {
			{
				Instance:    "chrome",
				Measurement: 123,
			},
		},
	}
	// unset
	{
		events := reader.groupToEvents(counters)
		assert.NotNil(t, events)
		assert.Equal(t, 1, len(events))

		ok, err := events[0].MetricSetFields.HasKey("object")
		assert.NoError(t, err)
		assert.True(t, ok)

		val, err := events[0].MetricSetFields.GetValue("object")
		assert.NoError(t, err)
		assert.Equal(t, "Process", val)
	}

	// ExtractObjectFromCounter = true
	reader.config.ExtractObjectFromCounter = &_true
	{
		events := reader.groupToEvents(counters)
		assert.NotNil(t, events)
		assert.Equal(t, 1, len(events))

		ok, err := events[0].MetricSetFields.HasKey("object")
		assert.NoError(t, err)
		assert.True(t, ok)

		val, err := events[0].MetricSetFields.GetValue("object")
		assert.NoError(t, err)
		assert.Equal(t, "Process(chrome)", val)
	}

	// ExtractObjectFromCounter = false
	reader.config.ExtractObjectFromCounter = &_false
	{
		events := reader.groupToEvents(counters)
		assert.NotNil(t, events)
		assert.Equal(t, 1, len(events))

		ok, err := events[0].MetricSetFields.HasKey("object")
		assert.NoError(t, err)
		assert.True(t, ok)

		val, err := events[0].MetricSetFields.GetValue("object")
		assert.NoError(t, err)
		assert.Equal(t, "Process", val)
	}
}
