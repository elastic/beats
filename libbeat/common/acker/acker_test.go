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

package acker

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v8/libbeat/beat"
)

type fakeACKer struct {
	AddEventFunc  func(event beat.Event, published bool)
	ACKEventsFunc func(n int)
	CloseFunc     func()
}

func TestNil(t *testing.T) {
	acker := Nil()
	require.NotNil(t, acker)

	// check acker can be used without panic:
	acker.AddEvent(beat.Event{}, false)
	acker.AddEvent(beat.Event{}, true)
	acker.ACKEvents(3)
	acker.Close()
}

func TestCounting(t *testing.T) {
	t.Run("ack count is passed through", func(t *testing.T) {
		var n int
		acker := RawCounting(func(acked int) { n = acked })
		acker.ACKEvents(3)
		require.Equal(t, 3, n)
	})
}

func TestTracking(t *testing.T) {
	t.Run("dropped event is acked immediately if empty", func(t *testing.T) {
		var acked, total int
		TrackingCounter(func(a, t int) { acked, total = a, t }).AddEvent(beat.Event{}, false)
		require.Equal(t, 0, acked)
		require.Equal(t, 1, total)
	})

	t.Run("no dropped events", func(t *testing.T) {
		var acked, total int
		acker := TrackingCounter(func(a, t int) { acked, total = a, t })
		acker.AddEvent(beat.Event{}, true)
		acker.AddEvent(beat.Event{}, true)
		acker.ACKEvents(2)
		require.Equal(t, 2, acked)
		require.Equal(t, 2, total)
	})

	t.Run("acking published includes dropped events in middle", func(t *testing.T) {
		var acked, total int
		acker := TrackingCounter(func(a, t int) { acked, total = a, t })
		acker.AddEvent(beat.Event{}, true)
		acker.AddEvent(beat.Event{}, false)
		acker.AddEvent(beat.Event{}, false)
		acker.AddEvent(beat.Event{}, true)
		acker.ACKEvents(2)
		require.Equal(t, 2, acked)
		require.Equal(t, 4, total)
	})

	t.Run("acking published includes dropped events at end of ACK interval", func(t *testing.T) {
		var acked, total int
		acker := TrackingCounter(func(a, t int) { acked, total = a, t })
		acker.AddEvent(beat.Event{}, true)
		acker.AddEvent(beat.Event{}, true)
		acker.AddEvent(beat.Event{}, false)
		acker.AddEvent(beat.Event{}, false)
		acker.AddEvent(beat.Event{}, true)
		acker.ACKEvents(2)
		require.Equal(t, 2, acked)
		require.Equal(t, 4, total)
	})

	t.Run("partial ACKs", func(t *testing.T) {
		var acked, total int
		acker := TrackingCounter(func(a, t int) { acked, total = a, t })
		acker.AddEvent(beat.Event{}, true)
		acker.AddEvent(beat.Event{}, true)
		acker.AddEvent(beat.Event{}, true)
		acker.AddEvent(beat.Event{}, true)
		acker.AddEvent(beat.Event{}, false)
		acker.AddEvent(beat.Event{}, true)
		acker.AddEvent(beat.Event{}, true)

		acker.ACKEvents(2)
		require.Equal(t, 2, acked)
		require.Equal(t, 2, total)

		acker.ACKEvents(2)
		require.Equal(t, 2, acked)
		require.Equal(t, 3, total)
	})
}

func TestEventPrivateReporter(t *testing.T) {
	t.Run("dropped event is acked immediately if empty", func(t *testing.T) {
		var acked int
		var data []interface{}
		acker := EventPrivateReporter(func(a int, d []interface{}) { acked, data = a, d })
		acker.AddEvent(beat.Event{Private: 1}, false)
		require.Equal(t, 0, acked)
		require.Equal(t, []interface{}{1}, data)
	})

	t.Run("no dropped events", func(t *testing.T) {
		var acked int
		var data []interface{}
		acker := EventPrivateReporter(func(a int, d []interface{}) { acked, data = a, d })
		acker.AddEvent(beat.Event{Private: 1}, true)
		acker.AddEvent(beat.Event{Private: 2}, true)
		acker.AddEvent(beat.Event{Private: 3}, true)
		acker.ACKEvents(3)
		require.Equal(t, 3, acked)
		require.Equal(t, []interface{}{1, 2, 3}, data)
	})

	t.Run("private of dropped events is included", func(t *testing.T) {
		var acked int
		var data []interface{}
		acker := EventPrivateReporter(func(a int, d []interface{}) { acked, data = a, d })
		acker.AddEvent(beat.Event{Private: 1}, true)
		acker.AddEvent(beat.Event{Private: 2}, false)
		acker.AddEvent(beat.Event{Private: 3}, true)
		acker.ACKEvents(2)
		require.Equal(t, 2, acked)
		require.Equal(t, []interface{}{1, 2, 3}, data)
	})
}

func TestLastEventPrivateReporter(t *testing.T) {
	t.Run("dropped event with private is acked immediately if empty", func(t *testing.T) {
		var acked int
		var datum interface{}
		acker := LastEventPrivateReporter(func(a int, d interface{}) { acked, datum = a, d })
		acker.AddEvent(beat.Event{Private: 1}, false)
		require.Equal(t, 0, acked)
		require.Equal(t, 1, datum)
	})

	t.Run("dropped event without private is ignored", func(t *testing.T) {
		var called bool
		acker := LastEventPrivateReporter(func(_ int, _ interface{}) { called = true })
		acker.AddEvent(beat.Event{Private: nil}, false)
		require.False(t, called)
	})

	t.Run("no dropped events", func(t *testing.T) {
		var acked int
		var data interface{}
		acker := LastEventPrivateReporter(func(a int, d interface{}) { acked, data = a, d })
		acker.AddEvent(beat.Event{Private: 1}, true)
		acker.AddEvent(beat.Event{Private: 2}, true)
		acker.AddEvent(beat.Event{Private: 3}, true)
		acker.ACKEvents(3)
		require.Equal(t, 3, acked)
		require.Equal(t, 3, data)
	})
}

func TestCombine(t *testing.T) {
	t.Run("AddEvent distributes", func(t *testing.T) {
		var a1, a2 int
		acker := Combine(countACKerOps(&a1, nil, nil), countACKerOps(&a2, nil, nil))
		acker.AddEvent(beat.Event{}, true)
		require.Equal(t, 1, a1)
		require.Equal(t, 1, a2)
	})

	t.Run("ACKEvents distributes", func(t *testing.T) {
		var a1, a2 int
		acker := Combine(countACKerOps(nil, &a1, nil), countACKerOps(nil, &a2, nil))
		acker.ACKEvents(1)
		require.Equal(t, 1, a1)
		require.Equal(t, 1, a2)
	})

	t.Run("Close distributes", func(t *testing.T) {
		var c1, c2 int
		acker := Combine(countACKerOps(nil, nil, &c1), countACKerOps(nil, nil, &c2))
		acker.Close()
		require.Equal(t, 1, c1)
		require.Equal(t, 1, c2)
	})
}

func TestConnectionOnly(t *testing.T) {
	t.Run("passes ACKs if not closed", func(t *testing.T) {
		var n int
		acker := ConnectionOnly(RawCounting(func(acked int) { n = acked }))
		acker.ACKEvents(3)
		require.Equal(t, 3, n)
	})

	t.Run("ignores ACKs after close", func(t *testing.T) {
		var n int
		acker := ConnectionOnly(RawCounting(func(acked int) { n = acked }))
		acker.Close()
		acker.ACKEvents(3)
		require.Equal(t, 0, n)
	})
}

func countACKerOps(add, acked, close *int) beat.ACKer {
	return &fakeACKer{
		AddEventFunc:  func(_ beat.Event, _ bool) { *add++ },
		ACKEventsFunc: func(_ int) { *acked++ },
		CloseFunc:     func() { *close++ },
	}
}

func (f *fakeACKer) AddEvent(event beat.Event, published bool) {
	if f.AddEventFunc != nil {
		f.AddEventFunc(event, published)
	}
}

func (f *fakeACKer) ACKEvents(n int) {
	if f.ACKEventsFunc != nil {
		f.ACKEventsFunc(n)
	}
}

func (f *fakeACKer) Close() {
	if f.CloseFunc != nil {
		f.CloseFunc()
	}
}
