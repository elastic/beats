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
// +build windows

package perfmon

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/metricbeat/helper/windows/pdh"
)

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
	assert.Equal(t, val, common.MapStr{"processor_count": float64(2)})
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
	assert.Equal(t, val, common.MapStr{"processor_count": float64(2)})
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
