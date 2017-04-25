// +build integration windows

package perfmon

import (
	"testing"
	"time"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const processorTimeCounter = `\Processor Information(_Total)\% Processor Time`

func TestQuery(t *testing.T) {
	q, err := NewQuery("")
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()

	err = q.AddCounter(processorTimeCounter)
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

	assert.NoError(t, value.Err)
}

func TestExistingCounter(t *testing.T) {
	config := make([]CounterConfig, 1)
	config[0].Alias = "processor.time.total.pct"
	config[0].Query = processorTimeCounter
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
	config := make([]CounterConfig, 1)
	config[0].Alias = "processor.time.total.pct"
	config[0].Query = "\\Processor Information(_Total)\\not existing counter"
	handle, err := NewPerfmonReader(config)
	if assert.Error(t, err) {
		assert.EqualValues(t, PDH_CSTATUS_NO_COUNTER, errors.Cause(err))
	}

	if handle != nil {
		err = handle.query.Close()
		assert.NoError(t, err)
	}
}

func TestNonExistingObject(t *testing.T) {
	config := make([]CounterConfig, 1)
	config[0].Alias = "processor.time.total.pct"
	config[0].Query = "\\non existing object\\% Processor Time"
	handle, err := NewPerfmonReader(config)
	if assert.Error(t, err) {
		assert.EqualValues(t, PDH_CSTATUS_NO_OBJECT, errors.Cause(err))
	}

	if handle != nil {
		err = handle.query.Close()
		assert.NoError(t, err)
	}
}

func TestRawValues(t *testing.T) {
	query, err := NewQuery("")
	if err != nil {
		t.Fatal(err)
	}
	defer query.Close()

	err = query.AddCounter(processorTimeCounter)
	if err != nil && err != PDH_NO_MORE_DATA {
		t.Fatal(err)
	}

	var values []float64

	for i := 0; i < 2; i++ {

		if err = query.Execute(); err != nil {
			t.Fatal(err)
		}

		_, rawvalue1, err := PdhGetRawCounterValue(query.counters[processorTimeCounter])

		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Millisecond * 1000)

		if err = query.Execute(); err != nil {
			t.Fatal(err)
		}

		_, rawvalue2, err := PdhGetRawCounterValue(query.counters[processorTimeCounter])

		if err != nil {
			t.Fatal(err)
		}

		value, err := PdhCalculateCounterFromRawValue(query.counters[processorTimeCounter], PdhFmtDouble|PdhFmtNoCap100, rawvalue2, rawvalue1)

		if err != nil {
			t.Fatal(err)
		}

		values = append(values, *(*float64)(unsafe.Pointer(&value.LongValue)))

	}

	t.Log(values)

}
