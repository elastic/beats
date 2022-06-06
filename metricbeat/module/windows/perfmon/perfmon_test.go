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

//go:build integration && windows
// +build integration,windows

package perfmon

import (
	"strings"
	"testing"
	"time"

	"github.com/elastic/beats/v7/metricbeat/helper/windows/pdh"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

const processorTimeCounter = `\Processor Information(_Total)\% Processor Time`

func TestData(t *testing.T) {
	config := map[string]interface{}{
		"module":     "windows",
		"metricsets": []string{"perfmon"},
		"perfmon.queries": []map[string]interface{}{
			{
				"object":   "Processor Information",
				"instance": []string{"_Total"},
				"counters": []map[string]interface{}{
					{
						"name":  "% Processor Time",
						"field": "processor.time.total.pct",
					},
					{
						"name": "% User Time",
					},
				},
			},
		},
	}

	ms := mbtest.NewReportingMetricSetV2Error(t, config)
	mbtest.ReportingFetchV2Error(ms)
	time.Sleep(60 * time.Millisecond)

	if err := mbtest.WriteEventsReporterV2Error(ms, t, "/"); err != nil {
		t.Fatal("write", err)
	}

}

func TestDataDeprecated(t *testing.T) {
	config := map[string]interface{}{
		"module":     "windows",
		"metricsets": []string{"perfmon"},
		"perfmon.counters": []map[string]string{
			{
				"instance_label":    "processor.name",
				"measurement_label": "processor.time.total.pct",
				"query":             `\Processor Information(_Total)\% Processor Time`,
			},
			{
				"instance_label":    "process.name",
				"measurement_label": "process.ID",
				"query":             `\Process(_Total)\ID Process`,
			},
			{
				"instance_label":    "processor.name",
				"measurement_label": "processor.time.user.ns",
				"query":             `\Processor Information(_Total)\% User Time`,
			},
		},
	}

	ms := mbtest.NewReportingMetricSetV2Error(t, config)
	mbtest.ReportingFetchV2Error(ms)
	time.Sleep(60 * time.Millisecond)

	events, errs := mbtest.ReportingFetchV2Error(ms)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	if len(events) == 0 {
		t.Fatal("no events received")
	}

	beatEvent := mbtest.StandardizeEvent(ms, events[0], mb.AddMetricSetInfo)
	mbtest.WriteEventToDataJSON(t, beatEvent, "")
}

func TestCounterWithNoInstanceName(t *testing.T) {
	config := map[string]interface{}{
		"module":     "windows",
		"metricsets": []string{"perfmon"},
		"perfmon.queries": []map[string]interface{}{
			{
				"object": "UDPv4",
				"counters": []map[string]interface{}{
					{
						"name": "Datagrams Sent/sec",
					},
				},
			},
		},
	}

	ms := mbtest.NewReportingMetricSetV2Error(t, config)
	mbtest.ReportingFetchV2Error(ms)
	time.Sleep(60 * time.Millisecond)

	events, errs := mbtest.ReportingFetchV2Error(ms)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	if len(events) == 0 {
		t.Fatal("no events received")
	}
	val, err := events[0].MetricSetFields.GetValue("object")
	assert.NoError(t, err)
	// Check values
	assert.EqualValues(t, "UDPv4", val)

}

func TestQuery(t *testing.T) {
	var q pdh.Query
	err := q.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()
	path, err := q.GetCounterPaths(processorTimeCounter)
	if err != nil {
		t.Fatal(err)
	}
	err = q.AddCounter(path[0], "TestInstanceName", "float", false)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		err = q.CollectData()
		if err != nil {
			t.Fatal(err)
		}
	}

	values, err := q.GetFormattedCounterValues()
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, values, 1)

	value, found := values[path[0]]
	if !found {
		t.Fatal(path[0], "not found")
	}

	assert.NoError(t, value[0].Err.Error)
	assert.Equal(t, "TestInstanceName", value[0].Instance)
}

func TestExistingCounter(t *testing.T) {
	config := Config{
		Queries: make([]Query, 1),
	}
	config.Queries[0].Name = "Processor Information"
	config.Queries[0].Instance = []string{"_Total"}
	config.Queries[0].Counters = []QueryCounter{
		{
			Name: "% Processor Time",
		},
	}
	handle, err := NewReader(config)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()

	values, err := handle.Read()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(values)
}

func TestExistingCounterDeprecated(t *testing.T) {
	config := Config{
		Counters: make([]Counter, 1),
	}
	config.Counters[0].InstanceLabel = "processor.name"
	config.Counters[0].MeasurementLabel = "processor.time.total.pct"
	config.Counters[0].Query = processorTimeCounter
	config.Counters[0].Format = "float"
	handle, err := NewReader(config)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.query.Close()

	values, err := handle.Read()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(values)
}

func TestNonExistingCounter(t *testing.T) {
	config := Config{
		Queries: make([]Query, 1),
	}
	config.Queries[0].Name = "Processor Information"
	config.Queries[0].Instance = []string{"_Total"}
	config.Queries[0].Counters = []QueryCounter{
		{
			Name: "% Processor Time time",
		},
	}
	handle, err := NewReader(config)
	if assert.Error(t, err) {
		assert.EqualValues(t, pdh.PDH_CSTATUS_NO_COUNTER, errors.Cause(err))
	}

	if handle != nil {
		err = handle.Close()
		assert.NoError(t, err)
	}
}

func TestNonExistingCounterDeprecated(t *testing.T) {
	config := Config{
		Counters: make([]Counter, 1),
	}
	config.Counters[0].InstanceLabel = "processor.name"
	config.Counters[0].MeasurementLabel = "processor.time.total.pct"
	config.Counters[0].Query = "\\Processor Information(_Total)\\not existing counter"
	config.Counters[0].Format = "float"
	handle, err := NewReader(config)
	if assert.Error(t, err) {
		assert.EqualValues(t, pdh.PDH_CSTATUS_NO_COUNTER, errors.Cause(err))
	}

	if handle != nil {
		err = handle.query.Close()
		assert.NoError(t, err)
	}
}

func TestIgnoreNonExistentCounter(t *testing.T) {
	config := Config{
		Queries:          make([]Query, 1),
		IgnoreNECounters: true,
	}
	config.Queries[0].Name = "Processor Information"
	config.Queries[0].Instance = []string{"_Total"}
	config.Queries[0].Counters = []QueryCounter{
		{
			Name: "% Processor Time time",
		},
	}
	handle, err := NewReader(config)

	values, err := handle.Read()

	if assert.Error(t, err) {
		assert.EqualValues(t, pdh.PDH_NO_DATA, errors.Cause(err))
	}

	if handle != nil {
		err = handle.Close()
		assert.NoError(t, err)
	}

	t.Log(values)
}

func TestIgnoreNonExistentCounterDeprecated(t *testing.T) {
	config := Config{
		Counters:         make([]Counter, 1),
		IgnoreNECounters: true,
	}
	config.Counters[0].InstanceLabel = "processor.name"
	config.Counters[0].MeasurementLabel = "processor.time.total.pct"
	config.Counters[0].Query = "\\Processor Information(_Total)\\not existing counter"
	config.Counters[0].Format = "float"
	handle, err := NewReader(config)

	values, err := handle.Read()

	if assert.Error(t, err) {
		assert.EqualValues(t, pdh.PDH_NO_DATA, errors.Cause(err))
	}

	if handle != nil {
		err = handle.query.Close()
		assert.NoError(t, err)
	}

	t.Log(values)
}

func TestNonExistingObject(t *testing.T) {
	config := Config{
		Queries: make([]Query, 1),
	}
	config.Queries[0].Name = "Processor MisInformation"
	config.Queries[0].Instance = []string{"_Total"}
	config.Queries[0].Counters = []QueryCounter{
		{
			Name: "% Processor Time",
		},
	}
	handle, err := NewReader(config)
	if assert.Error(t, err) {
		assert.EqualValues(t, pdh.PDH_CSTATUS_NO_OBJECT, errors.Cause(err))
	}

	if handle != nil {
		err = handle.Close()
		assert.NoError(t, err)
	}
}

func TestNonExistingObjectDeprecated(t *testing.T) {
	config := Config{
		Counters: make([]Counter, 1),
	}
	config.Counters[0].InstanceLabel = "processor.name"
	config.Counters[0].MeasurementLabel = "processor.time.total.pct"
	config.Counters[0].Query = "\\non existing object\\% Processor Performance"
	config.Counters[0].Format = "float"
	handle, err := NewReader(config)
	if assert.Error(t, err) {
		assert.EqualValues(t, pdh.PDH_CSTATUS_NO_OBJECT, errors.Cause(err))
	}

	if handle != nil {
		err = handle.query.Close()
		assert.NoError(t, err)
	}
}

func TestLongOutputFormat(t *testing.T) {
	var query pdh.Query
	err := query.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer query.Close()
	path, err := query.GetCounterPaths(processorTimeCounter)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotZero(t, len(path))
	err = query.AddCounter(path[0], "", "long", false)
	if err != nil && err != pdh.PDH_NO_MORE_DATA {
		t.Fatal(err)
	}

	err = query.CollectData()
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 1000)

	err = query.CollectData()
	if err != nil {
		t.Fatal(err)
	}

	values, err := query.GetFormattedCounterValues()
	if err != nil {
		t.Fatal(err)
	}

	_, okLong := values[path[0]][0].Measurement.(int32)

	assert.True(t, okLong)
}

func TestFloatOutputFormat(t *testing.T) {
	var query pdh.Query
	err := query.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer query.Close()
	path, err := query.GetCounterPaths(processorTimeCounter)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotZero(t, len(path))
	err = query.AddCounter(path[0], "", "float", false)
	if err != nil && err != pdh.PDH_NO_MORE_DATA {
		t.Fatal(err)
	}

	err = query.CollectData()
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 1000)

	err = query.CollectData()
	if err != nil {
		t.Fatal(err)
	}

	values, err := query.GetFormattedCounterValues()
	if err != nil {
		t.Fatal(err)
	}

	_, okFloat := values[path[0]][0].Measurement.(float64)

	assert.True(t, okFloat)
}

func TestWildcardQuery(t *testing.T) {
	config := Config{
		Queries: make([]Query, 1),
	}
	config.Queries[0].Name = "Processor Information"
	config.Queries[0].Instance = []string{"*"}
	config.Queries[0].Namespace = "metrics"
	config.Queries[0].Counters = []QueryCounter{
		{
			Name: "% Processor Time",
		},
	}
	handle, err := NewReader(config)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()

	values, _ := handle.Read()

	time.Sleep(time.Millisecond * 1000)

	values, err = handle.Read()
	if err != nil {
		t.Fatal(err)
	}
	assert.NotZero(t, len(values))
	pctKey, err := values[0].MetricSetFields.HasKey("metrics.%_processor_time")
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, pctKey)
	t.Log(values)
}

func TestWildcardQueryNoInstanceName(t *testing.T) {
	config := Config{
		Queries: make([]Query, 1),
	}
	config.Queries[0].Name = "Process"
	config.Queries[0].Instance = []string{"*"}
	config.Queries[0].Namespace = "metrics"
	config.Queries[0].Counters = []QueryCounter{
		{
			Name: "Private Bytes",
		},
	}

	handle, err := NewReader(config)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()

	values, _ := handle.Read()

	time.Sleep(time.Millisecond * 1000)

	values, err = handle.Read()
	if err != nil {
		t.Fatal(err)
	}
	assert.NotZero(t, len(values))
	pctKey, err := values[0].MetricSetFields.HasKey("metrics.private_bytes")
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, pctKey)

	for _, s := range values {
		instance, err := s.MetricSetFields.GetValue("instance")
		if err != nil {
			t.Fatal(err)
		}
		assert.False(t, strings.Contains(instance.(string), "*"))
	}

	t.Log(values)
}

func TestGroupByInstance(t *testing.T) {
	config := Config{
		Queries:           make([]Query, 1),
		GroupMeasurements: true,
	}
	config.Queries[0].Name = "Processor Information"
	config.Queries[0].Instance = []string{"_Total"}
	config.Queries[0].Namespace = "metrics"
	config.Queries[0].Counters = []QueryCounter{
		{
			Name: "% Processor Time",
		},
		{
			Name: "% User Time",
		},
		{
			Name: "% Privileged Time",
		},
	}
	handle, err := NewReader(config)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()

	values, _ := handle.Read()

	time.Sleep(time.Millisecond * 1000)

	values, err = handle.Read()
	if err != nil {
		t.Fatal(err)
	}

	assert.EqualValues(t, 1, len(values)) // Assert all metrics have been grouped into a single event

	// Test all keys exist in the event
	pctKey, err := values[0].MetricSetFields.HasKey("metrics.%_processor_time")
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, pctKey)

	pctKey, err = values[0].MetricSetFields.HasKey("metrics.%_user_time")
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, pctKey)

	pctKey, err = values[0].MetricSetFields.HasKey("metrics.%_privileged_time")
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, pctKey)

	t.Log(values)
}

func TestGroupByInstanceDeprecated(t *testing.T) {
	config := Config{
		Counters:          make([]Counter, 3),
		GroupMeasurements: true,
	}
	config.Counters[0].InstanceLabel = "processor.name"
	config.Counters[0].MeasurementLabel = "processor.time.pct"
	config.Counters[0].Query = `\Processor Information(_Total)\% Processor Time`
	config.Counters[0].Format = "float"

	config.Counters[1].InstanceLabel = "processor.name"
	config.Counters[1].MeasurementLabel = "processor.time.user.pct"
	config.Counters[1].Query = `\Processor Information(_Total)\% User Time`
	config.Counters[1].Format = "float"

	config.Counters[2].InstanceLabel = "processor.name"
	config.Counters[2].MeasurementLabel = "processor.time.privileged.ns"
	config.Counters[2].Query = `\Processor Information(_Total)\% Privileged Time`
	config.Counters[2].Format = "float"

	handle, err := NewReader(config)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.query.Close()

	values, _ := handle.Read()

	time.Sleep(time.Millisecond * 1000)

	values, err = handle.Read()
	if err != nil {
		t.Fatal(err)
	}

	assert.EqualValues(t, 1, len(values)) // Assert all metrics have been grouped into a single event

	// Test all keys exist in the event
	pctKey, err := values[0].MetricSetFields.HasKey("processor.time.pct")
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, pctKey)

	pctKey, err = values[0].MetricSetFields.HasKey("processor.time.user.pct")
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, pctKey)

	pctKey, err = values[0].MetricSetFields.HasKey("processor.time.privileged.ns")
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, pctKey)

	t.Log(values)
}
