// +build integration windows

package perfmon

import (
	"testing"
	"time"
	"unsafe"

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
				"instance_label":    "disk.bytes.name",
				"measurement_label": "disk.bytes.read.total",
				"query":             `\FileSystem Disk Activity(_Total)\FileSystem Bytes Read`,
			},
			{
				"instance_label":    "processor.name",
				"measurement_label": "processor.time.idle.average.ns",
				"query":             `\Processor Information(_Total)\Average Idle Time`,
			},
		},
	}

	ms := mbtest.NewReportingMetricSetV2(t, config)
	mbtest.ReportingFetchV2(ms)
	time.Sleep(60 * time.Millisecond)

	events, errs := mbtest.ReportingFetchV2(ms)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	if len(events) == 0 {
		t.Fatal("no events received")
	}

	beatEvent := mbtest.StandardizeEvent(ms, events[0], mb.AddMetricSetInfo)
	mbtest.WriteEventToDataJSON(t, beatEvent)
}

func TestQuery(t *testing.T) {
	q, err := NewQuery("")
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()

	err = q.AddCounter(processorTimeCounter, FloatFlormat, "")
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		err = q.Execute()
		if err != nil {
			t.Fatal(err)
		}
	}

	values, err := q.Values()
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, values, 1)

	value, found := values[processorTimeCounter]
	if !found {
		t.Fatal(processorTimeCounter, "not found")
	}

	assert.NoError(t, value[0].Err)
}

func TestExistingCounter(t *testing.T) {
	config := Config{
		CounterConfig: make([]CounterConfig, 1),
	}
	config.CounterConfig[0].InstanceLabel = "processor.name"
	config.CounterConfig[0].MeasurementLabel = "processor.time.total.pct"
	config.CounterConfig[0].Query = processorTimeCounter
	config.CounterConfig[0].Format = "float"
	handle, err := NewPerfmonReader(config)
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
	handle, err := NewPerfmonReader(config)
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
	handle, err := NewPerfmonReader(config)

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
	handle, err := NewPerfmonReader(config)
	if assert.Error(t, err) {
		assert.EqualValues(t, PDH_CSTATUS_NO_OBJECT, errors.Cause(err))
	}

	if handle != nil {
		err = handle.query.Close()
		assert.NoError(t, err)
	}
}

func TestLongOutputFormat(t *testing.T) {
	query, err := NewQuery("")
	if err != nil {
		t.Fatal(err)
	}
	defer query.Close()

	err = query.AddCounter(processorTimeCounter, LongFormat, "")
	if err != nil && err != PDH_NO_MORE_DATA {
		t.Fatal(err)
	}

	err = query.Execute()
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 1000)

	err = query.Execute()
	if err != nil {
		t.Fatal(err)
	}

	values, err := query.Values()
	if err != nil {
		t.Fatal(err)
	}

	_, okLong := values[processorTimeCounter][0].Measurement.(int64)

	assert.True(t, okLong)
}

func TestFloatOutputFormat(t *testing.T) {
	query, err := NewQuery("")
	if err != nil {
		t.Fatal(err)
	}
	defer query.Close()

	err = query.AddCounter(processorTimeCounter, FloatFlormat, "")
	if err != nil && err != PDH_NO_MORE_DATA {
		t.Fatal(err)
	}

	err = query.Execute()
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 1000)

	err = query.Execute()
	if err != nil {
		t.Fatal(err)
	}

	values, err := query.Values()
	if err != nil {
		t.Fatal(err)
	}

	_, okFloat := values[processorTimeCounter][0].Measurement.(float64)

	assert.True(t, okFloat)
}

func TestRawValues(t *testing.T) {
	query, err := NewQuery("")
	if err != nil {
		t.Fatal(err)
	}
	defer query.Close()

	err = query.AddCounter(processorTimeCounter, FloatFlormat, "")
	if err != nil && err != PDH_NO_MORE_DATA {
		t.Fatal(err)
	}

	var values []float64

	for i := 0; i < 2; i++ {
		if err = query.Execute(); err != nil {
			t.Fatal(err)
		}

		_, rawvalue1, err := PdhGetRawCounterValue(query.counters[processorTimeCounter].handle)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Millisecond * 1000)

		if err = query.Execute(); err != nil {
			t.Fatal(err)
		}

		_, rawvalue2, err := PdhGetRawCounterValue(query.counters[processorTimeCounter].handle)
		if err != nil {
			t.Fatal(err)
		}

		value, err := PdhCalculateCounterFromRawValue(query.counters[processorTimeCounter].handle, PdhFmtDouble|PdhFmtNoCap100, rawvalue2, rawvalue1)
		if err != nil {
			t.Fatal(err)
		}

		values = append(values, *(*float64)(unsafe.Pointer(&value.LongValue)))
	}

	t.Log(values)
}

func TestWildcardQuery(t *testing.T) {
	config := Config{
		CounterConfig: make([]CounterConfig, 1),
	}
	config.CounterConfig[0].InstanceLabel = "processor.name"
	config.CounterConfig[0].MeasurementLabel = "processor.time.pct"
	config.CounterConfig[0].Query = `\Processor Information(*)\% Processor Time`
	config.CounterConfig[0].Format = "float"
	handle, err := NewPerfmonReader(config)
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

	t.Log(values)
}
