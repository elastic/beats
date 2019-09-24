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

// +build integration windows

package perfmon

import (
	"strings"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/metricbeat/mb"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

const processorTimeCounter = `\Processor Information(_Total)\% Processor Time`

func TestData(t *testing.T) {
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
		"perfmon.counters": []map[string]string{
			{
				"instance_label":    "processor.name",
				"measurement_label": "processor.time.total.pct",
				"query":             `\UDPv4\Datagrams Sent/sec`,
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
	process := events[0].MetricSetFields["processor"].(common.MapStr)
	// Check values
	assert.EqualValues(t, "UDPv4", process["name"])

}

func TestQuery(t *testing.T) {
	var q Query
	err := q.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()
	counter := CounterConfig{Format: "float", InstanceName: "TestInstanceName"}
	err = q.AddCounter(processorTimeCounter, counter, false)
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

	value, found := values[processorTimeCounter]
	if !found {
		t.Fatal(processorTimeCounter, "not found")
	}

	assert.NoError(t, value[0].Err)
	assert.Equal(t, "TestInstanceName", value[0].Instance)
}

func TestExistingCounter(t *testing.T) {
	config := Config{
		CounterConfig: make([]CounterConfig, 1),
	}
	config.CounterConfig[0].InstanceLabel = "processor.name"
	config.CounterConfig[0].MeasurementLabel = "processor.time.total.pct"
	config.CounterConfig[0].Query = processorTimeCounter
	config.CounterConfig[0].Format = "float"
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
		CounterConfig: make([]CounterConfig, 1),
	}
	config.CounterConfig[0].InstanceLabel = "processor.name"
	config.CounterConfig[0].MeasurementLabel = "processor.time.total.pct"
	config.CounterConfig[0].Query = "\\Processor Information(_Total)\\not existing counter"
	config.CounterConfig[0].Format = "float"
	handle, err := NewReader(config)
	if assert.Error(t, err) {
		assert.EqualValues(t, PDH_CSTATUS_NO_COUNTER, errors.Cause(err))
	}

	if handle != nil {
		err = handle.query.Close()
		assert.NoError(t, err)
	}
}

func TestIgnoreNonExistentCounter(t *testing.T) {
	config := Config{
		CounterConfig:    make([]CounterConfig, 1),
		IgnoreNECounters: true,
	}
	config.CounterConfig[0].InstanceLabel = "processor.name"
	config.CounterConfig[0].MeasurementLabel = "processor.time.total.pct"
	config.CounterConfig[0].Query = "\\Processor Information(_Total)\\not existing counter"
	config.CounterConfig[0].Format = "float"
	handle, err := NewReader(config)

	values, err := handle.Read()

	if assert.Error(t, err) {
		assert.EqualValues(t, PDH_NO_DATA, errors.Cause(err))
	}

	if handle != nil {
		err = handle.query.Close()
		assert.NoError(t, err)
	}

	t.Log(values)
}

func TestNonExistingObject(t *testing.T) {
	config := Config{
		CounterConfig: make([]CounterConfig, 1),
	}
	config.CounterConfig[0].InstanceLabel = "processor.name"
	config.CounterConfig[0].MeasurementLabel = "processor.time.total.pct"
	config.CounterConfig[0].Query = "\\non existing object\\% Processor Performance"
	config.CounterConfig[0].Format = "float"
	handle, err := NewReader(config)
	if assert.Error(t, err) {
		assert.EqualValues(t, PDH_CSTATUS_NO_OBJECT, errors.Cause(err))
	}

	if handle != nil {
		err = handle.query.Close()
		assert.NoError(t, err)
	}
}

func TestLongOutputFormat(t *testing.T) {
	var query Query
	err := query.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer query.Close()
	counter := CounterConfig{Format: "long"}
	err = query.AddCounter(processorTimeCounter, counter, false)
	if err != nil && err != PDH_NO_MORE_DATA {
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

	_, okLong := values[processorTimeCounter][0].Measurement.(int32)

	assert.True(t, okLong)
}

func TestFloatOutputFormat(t *testing.T) {
	var query Query
	err := query.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer query.Close()
	counter := CounterConfig{Format: "float"}
	err = query.AddCounter(processorTimeCounter, counter, false)
	if err != nil && err != PDH_NO_MORE_DATA {
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

	_, okFloat := values[processorTimeCounter][0].Measurement.(float64)

	assert.True(t, okFloat)
}

func TestWildcardQuery(t *testing.T) {
	config := Config{
		CounterConfig: make([]CounterConfig, 1),
	}
	config.CounterConfig[0].InstanceLabel = "processor.name"
	config.CounterConfig[0].InstanceName = "TestInstanceName"
	config.CounterConfig[0].MeasurementLabel = "processor.time.pct"
	config.CounterConfig[0].Query = `\Processor Information(*)\% Processor Time`
	config.CounterConfig[0].Format = "float"
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

	pctKey, err := values[0].MetricSetFields.HasKey("processor.time.pct")
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, pctKey)

	pct, err := values[0].MetricSetFields.GetValue("processor.name")
	if err != nil {
		t.Fatal(err)
	}
	assert.NotEqual(t, "TestInstanceName", pct)

	t.Log(values)
}

func TestWildcardQueryNoInstanceName(t *testing.T) {
	config := Config{
		CounterConfig: make([]CounterConfig, 1),
	}
	config.CounterConfig[0].InstanceLabel = "process_private_bytes"
	config.CounterConfig[0].MeasurementLabel = "process.private.bytes"
	config.CounterConfig[0].Query = `\Process(*)\Private Bytes`
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

	pctKey, err := values[0].MetricSetFields.HasKey("process.private.bytes")
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, pctKey)

	for _, s := range values {
		pct, err := s.MetricSetFields.GetValue("process_private_bytes")
		if err != nil {
			t.Fatal(err)
		}
		assert.False(t, strings.Contains(pct.(string), "*"))
	}

	t.Log(values)
}

func TestGroupByInstance(t *testing.T) {
	config := Config{
		CounterConfig:     make([]CounterConfig, 3),
		GroupMeasurements: true,
	}
	config.CounterConfig[0].InstanceLabel = "processor.name"
	config.CounterConfig[0].MeasurementLabel = "processor.time.pct"
	config.CounterConfig[0].Query = `\Processor Information(_Total)\% Processor Time`
	config.CounterConfig[0].Format = "float"

	config.CounterConfig[1].InstanceLabel = "processor.name"
	config.CounterConfig[1].MeasurementLabel = "processor.time.user.pct"
	config.CounterConfig[1].Query = `\Processor Information(_Total)\% User Time`
	config.CounterConfig[1].Format = "float"

	config.CounterConfig[2].InstanceLabel = "processor.name"
	config.CounterConfig[2].MeasurementLabel = "processor.time.privileged.ns"
	config.CounterConfig[2].Query = `\Processor Information(_Total)\% Privileged Time`
	config.CounterConfig[2].Format = "float"

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
