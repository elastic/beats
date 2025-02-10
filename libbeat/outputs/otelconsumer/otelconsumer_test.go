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

package otelconsumer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestPublish(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	event1 := beat.Event{Fields: mapstr.M{"field": 1}}
	event2 := beat.Event{Fields: mapstr.M{"field": 2}}
	event3 := beat.Event{Fields: mapstr.M{"field": 3}}

	makeOtelConsumer := func(t *testing.T, consumeFn func(ctx context.Context, ld plog.Logs) error) *otelConsumer {
		t.Helper()

		assert.NoError(t, logp.TestingSetup(logp.WithSelectors("otelconsumer")))

		logConsumer, err := consumer.NewLogs(consumeFn)
		assert.NoError(t, err)
		consumer := &otelConsumer{
			observer:     outputs.NewNilObserver(),
			logsConsumer: logConsumer,
			beatInfo:     beat.Info{},
			log:          logp.NewLogger("otelconsumer"),
		}
		return consumer
	}

	t.Run("ack batch on consumer success", func(t *testing.T) {
		batch := outest.NewBatch(event1, event2, event3)

		var countLogs int
		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			countLogs = countLogs + ld.LogRecordCount()
			return nil
		})

		err := otelConsumer.Publish(ctx, batch)
		assert.NoError(t, err)
		assert.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchACK, batch.Signals[0].Tag)
		assert.Equal(t, len(batch.Events()), countLogs, "all events should be consumed")
	})

	t.Run("data_stream fields are set on logrecord.Attribute", func(t *testing.T) {
		dataStreamField := mapstr.M{
			"type":      "logs",
			"namespace": "not_default",
			"dataset":   "not_elastic_agent",
		}
		event1.Fields["data_stream"] = dataStreamField

		batch := outest.NewBatch(event1)

		var countLogs int
		var attributes pcommon.Map
		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			countLogs = countLogs + ld.LogRecordCount()
			for i := 0; i < ld.ResourceLogs().Len(); i++ {
				resourceLog := ld.ResourceLogs().At(i)
				for j := 0; j < resourceLog.ScopeLogs().Len(); j++ {
					scopeLog := resourceLog.ScopeLogs().At(j)
					for k := 0; k < scopeLog.LogRecords().Len(); k++ {
						LogRecord := scopeLog.LogRecords().At(k)
						attributes = LogRecord.Attributes()
					}
				}
			}
			return nil
		})

		err := otelConsumer.Publish(ctx, batch)
		assert.NoError(t, err)
		assert.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchACK, batch.Signals[0].Tag)
		pcommonFields := mapstrToPcommonMap(event1.Fields)
		want, ok := pcommonFields.Get("data_stream")
		require.True(t, ok)
		got, ok := attributes.Get("data_stream")
		require.True(t, ok)
		assert.EqualValues(t, want.AsRaw(), got.AsRaw())
	})

	t.Run("retries the batch on non-permanent consumer error", func(t *testing.T) {
		batch := outest.NewBatch(event1, event2, event3)

		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			return errors.New("consume error")
		})

		err := otelConsumer.Publish(ctx, batch)
		assert.Error(t, err)
		assert.False(t, consumererror.IsPermanent(err))
		assert.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchRetry, batch.Signals[0].Tag)
	})

	t.Run("drop batch on permanent consumer error", func(t *testing.T) {
		batch := outest.NewBatch(event1, event2, event3)

		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			return consumererror.NewPermanent(errors.New("consumer error"))
		})

		err := otelConsumer.Publish(ctx, batch)
		assert.Error(t, err)
		assert.True(t, consumererror.IsPermanent(err))
		assert.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchDrop, batch.Signals[0].Tag)
	})

	t.Run("retries on context cancelled", func(t *testing.T) {
		batch := outest.NewBatch(event1, event2, event3)

		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			return context.Canceled
		})

		err := otelConsumer.Publish(ctx, batch)
		assert.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchRetry, batch.Signals[0].Tag)
	})
}

func TestMapstrToPcommonMapString(t *testing.T) {
	tests := map[string]struct {
		mapstr_val  interface{}
		pcommon_val string
	}{
		"forty two": {mapstr_val: "forty two", pcommon_val: "forty two"},
		"empty":     {mapstr_val: "", pcommon_val: ""},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			a := mapstr.M{"test": tc.mapstr_val}
			want := pcommon.NewMap()
			want.PutStr("test", tc.pcommon_val)
			got := mapstrToPcommonMap(a)
			assert.Equal(t, want, got)
		})
	}
}

func TestMapstrToPcommonMapSliceString(t *testing.T) {
	inputSlice := []string{"1", "2", "3"}
	inputMap := mapstr.M{
		"slice": inputSlice,
	}
	want := pcommon.NewMap()
	sliceOfInt := want.PutEmptySlice("slice")
	for _, i := range inputSlice {
		val := sliceOfInt.AppendEmpty()
		val.SetStr(i)
	}

	got := mapstrToPcommonMap(inputMap)
	assert.Equal(t, want, got)
}

func TestMapstrToPcommonMapInt(t *testing.T) {
	tests := map[string]struct {
		mapstr_val  interface{}
		pcommon_val int
	}{
		"int":    {mapstr_val: int(42), pcommon_val: 42},
		"int8":   {mapstr_val: int8(42), pcommon_val: 42},
		"int16":  {mapstr_val: int16(42), pcommon_val: 42},
		"int32":  {mapstr_val: int32(42), pcommon_val: 42},
		"int64":  {mapstr_val: int32(42), pcommon_val: 42},
		"uint":   {mapstr_val: uint(42), pcommon_val: 42},
		"uint8":  {mapstr_val: uint8(42), pcommon_val: 42},
		"uint16": {mapstr_val: uint16(42), pcommon_val: 42},
		"uint32": {mapstr_val: uint32(42), pcommon_val: 42},
		"uint64": {mapstr_val: uint64(42), pcommon_val: 42},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			a := mapstr.M{"test": tc.mapstr_val}
			want := pcommon.NewMap()
			want.PutInt("test", int64(tc.pcommon_val))
			got := mapstrToPcommonMap(a)
			assert.Equal(t, want, got)
		})
	}
}

func TestMapstrToPcommonMapSliceInt(t *testing.T) {
	inputSlice := []int{42, 43, 44}
	inputMap := mapstr.M{
		"slice": inputSlice,
	}
	want := pcommon.NewMap()
	sliceOfInt := want.PutEmptySlice("slice")
	for _, i := range inputSlice {
		val := sliceOfInt.AppendEmpty()
		val.SetInt(int64(i))
	}

	got := mapstrToPcommonMap(inputMap)
	assert.Equal(t, want, got)
}

func TestMapstrToPcommonMapDouble(t *testing.T) {
	tests := map[string]struct {
		mapstr_val  interface{}
		pcommon_val float64
	}{
		"float32": {mapstr_val: float32(4.2), pcommon_val: float64(float32(4.2))},
		"float64": {mapstr_val: float64(4.2), pcommon_val: 4.2},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			a := mapstr.M{"test": tc.mapstr_val}
			want := pcommon.NewMap()
			want.PutDouble("test", tc.pcommon_val)
			got := mapstrToPcommonMap(a)
			assert.Equal(t, want, got)
		})
	}
}

func TestMapstrToPcommonMapSliceDouble(t *testing.T) {
	inputSlice := []float32{4.2, 4.3, 4.4}
	inputMap := mapstr.M{
		"slice": inputSlice,
	}
	want := pcommon.NewMap()
	sliceOfInt := want.PutEmptySlice("slice")
	for _, i := range inputSlice {
		val := sliceOfInt.AppendEmpty()
		val.SetDouble(float64(i))
	}

	got := mapstrToPcommonMap(inputMap)
	assert.Equal(t, want, got)
}

func TestMapstrToPcommonMapBool(t *testing.T) {
	tests := map[string]struct {
		mapstr_val  interface{}
		pcommon_val bool
	}{
		"true":  {mapstr_val: true, pcommon_val: true},
		"false": {mapstr_val: false, pcommon_val: false},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			a := mapstr.M{"test": tc.mapstr_val}
			want := pcommon.NewMap()
			want.PutBool("test", tc.pcommon_val)
			got := mapstrToPcommonMap(a)
			assert.Equal(t, want, got)
		})
	}
}

func TestMapstrToPcommonMapSliceBool(t *testing.T) {
	inputSlice := []bool{true, false, true}
	inputMap := mapstr.M{
		"slice": inputSlice,
	}
	want := pcommon.NewMap()
	pcommonSlice := want.PutEmptySlice("slice")
	for _, i := range inputSlice {
		val := pcommonSlice.AppendEmpty()
		val.SetBool(i)
	}

	got := mapstrToPcommonMap(inputMap)
	assert.Equal(t, want, got)
}

func TestMapstrToPcommonMapMapstr(t *testing.T) {
	input := mapstr.M{
		"inner": mapstr.M{
			"inner_int": 42,
		},
	}
	want := pcommon.NewMap()
	inner := want.PutEmptyMap("inner")
	inner.PutInt("inner_int", 42)

	got := mapstrToPcommonMap(input)
	assert.Equal(t, want, got)
}

func TestMapstrToPcommonMapSliceMapstr(t *testing.T) {
	inputSlice := []mapstr.M{mapstr.M{"item": 1}, mapstr.M{"item": 1}, mapstr.M{"item": 1}}
	inputMap := mapstr.M{
		"slice": inputSlice,
	}
	want := pcommon.NewMap()
	sliceOfInt := want.PutEmptySlice("slice")
	for range inputSlice {
		val := sliceOfInt.AppendEmpty()
		newMap := pcommon.NewMap()
		newMap.PutInt("item", 1)
		newMap.CopyTo(val.SetEmptyMap())
	}

	got := mapstrToPcommonMap(inputMap)
	assert.Equal(t, want, got)
}

func TestMapstrToPcommonMapSliceTime(t *testing.T) {
	times := []struct {
		mapstr_val  string
		pcommon_val int64
	}{
		{mapstr_val: "2006-01-02T15:04:05+07:00", pcommon_val: 1136189045000},
		{mapstr_val: "1970-01-01T00:00:00+00:00", pcommon_val: 0},
	}
	var sliceTimes []time.Time
	pcommonSlice := pcommon.NewSlice()
	for _, tc := range times {
		targetTime, err := time.Parse(time.RFC3339, tc.mapstr_val)
		assert.NoError(t, err, "Error parsing time")
		sliceTimes = append(sliceTimes, targetTime)
		pVal := pcommonSlice.AppendEmpty()
		pVal.SetInt(tc.pcommon_val)
	}
	inputMap := mapstr.M{
		"slice": sliceTimes,
	}
	want := pcommon.NewMap()
	pcommonSlice.CopyTo(want.PutEmptySlice("slice"))
	got := mapstrToPcommonMap(inputMap)
	assert.Equal(t, want, got)
}
