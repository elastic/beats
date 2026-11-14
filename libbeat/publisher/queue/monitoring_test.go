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

package queue

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/monitoring"
)

func queueFilledPct(t *testing.T, reg *monitoring.Registry) float64 {
	t.Helper()
	snapshot := monitoring.CollectFlatSnapshot(reg, monitoring.Full, false)
	pct, ok := snapshot.Floats["queue.filled.pct"]
	require.True(t, ok, "queue.filled.pct should be present in the monitoring snapshot")
	return pct
}

func TestQueueObserverFilledPctUnlimitedQueue(t *testing.T) {
	// A disk queue configured with max_size: 0 is unlimited: it reports
	// MaxBytes(0) and never sets a maximum event count. The filled
	// percentage must stay finite instead of dividing by zero.
	reg := monitoring.NewRegistry()
	observer := NewQueueObserver(reg)
	observer.MaxBytes(0)

	observer.AddEvent(100)
	pct := queueFilledPct(t, reg)
	assert.False(t, math.IsInf(pct, 0), "filled.pct must not be infinite, got %v", pct)
	assert.False(t, math.IsNaN(pct), "filled.pct must not be NaN")
	assert.Equal(t, 0.0, pct)
}

func TestQueueObserverFilledPctUnlimitedEmptyQueue(t *testing.T) {
	// An empty unlimited queue computes 0 filled / 0 max, which must
	// not produce NaN.
	reg := monitoring.NewRegistry()
	observer := NewQueueObserver(reg)
	observer.MaxBytes(0)

	observer.Restore(0, 0)
	pct := queueFilledPct(t, reg)
	assert.False(t, math.IsNaN(pct), "filled.pct must not be NaN")
	assert.Equal(t, 0.0, pct)
}

func TestQueueObserverFilledPctFiniteLimits(t *testing.T) {
	// A finite byte limit (disk queue with a finite max_size) keeps
	// reporting the byte-based percentage.
	reg := monitoring.NewRegistry()
	observer := NewQueueObserver(reg)
	observer.MaxBytes(1000)
	observer.AddEvent(250)
	assert.Equal(t, 0.25, queueFilledPct(t, reg))

	// A finite event limit (memory queue) keeps reporting the
	// event-based percentage.
	reg = monitoring.NewRegistry()
	observer = NewQueueObserver(reg)
	observer.MaxEvents(4)
	observer.AddEvent(10)
	observer.AddEvent(10)
	assert.Equal(t, 0.5, queueFilledPct(t, reg))
}
