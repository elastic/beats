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
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// Observer is an interface for queues to send state updates to a metrics
// or test listener.
type Observer interface {
	MaxEvents(int)
	MaxBytes(int)

	// Restore queue state on startup. Used by the disk queue to report events
	// that are already in the queue from a previous run.
	Restore(eventCount int, byteCount int)

	// All reported byte counts are zero if the output doesn't support
	// early encoding.
	AddEvent(byteCount int)
	ConsumeEvents(eventCount int, byteCount int)
	RemoveEvents(eventCount int, byteCount int)
}

type queueObserver struct {
	maxEvents *monitoring.Uint // gauge
	maxBytes  *monitoring.Uint // gauge

	addedEvents    *monitoring.Uint
	addedBytes     *monitoring.Uint
	consumedEvents *monitoring.Uint
	consumedBytes  *monitoring.Uint
	removedEvents  *monitoring.Uint
	removedBytes   *monitoring.Uint

	filledEvents *monitoring.Uint  // gauge
	filledBytes  *monitoring.Uint  // gauge
	filledPct    *monitoring.Float // gauge

	// backwards compatibility: the metric "acked" is the old name for
	// "removed.events". Ideally we would like to define an alias in the
	// monitoring API, but until that's possible we shadow it with this
	// extra variable and make sure to always change removedEvents and
	// acked at the same time.
	acked *monitoring.Uint
}

type nilObserver struct{}

// Creates queue metrics in the given registry under the path "pipeline.queue".
func NewQueueObserver(metrics *monitoring.Registry) Observer {
	if metrics == nil {
		return nilObserver{}
	}
	queueMetrics := metrics.GetRegistry("queue")
	if queueMetrics != nil {
		err := queueMetrics.Clear()
		if err != nil {
			return nilObserver{}
		}
	} else {
		queueMetrics = metrics.NewRegistry("queue")
	}

	ob := &queueObserver{
		maxEvents: monitoring.NewUint(queueMetrics, "max_events"), // gauge
		maxBytes:  monitoring.NewUint(queueMetrics, "max_bytes"),  // gauge

		addedEvents:    monitoring.NewUint(queueMetrics, "added.events"),
		addedBytes:     monitoring.NewUint(queueMetrics, "added.bytes"),
		consumedEvents: monitoring.NewUint(queueMetrics, "consumed.events"),
		consumedBytes:  monitoring.NewUint(queueMetrics, "consumed.bytes"),
		removedEvents:  monitoring.NewUint(queueMetrics, "removed.events"),
		removedBytes:   monitoring.NewUint(queueMetrics, "removed.bytes"),

		filledEvents: monitoring.NewUint(queueMetrics, "filled.events"), // gauge
		filledBytes:  monitoring.NewUint(queueMetrics, "filled.bytes"),  // gauge
		filledPct:    monitoring.NewFloat(queueMetrics, "filled.pct"),   // gauge

		// backwards compatibility: "acked" is an alias for "removed.events".
		acked: monitoring.NewUint(queueMetrics, "acked"),
	}
	return ob
}

func (ob *queueObserver) MaxEvents(value int) {
	ob.maxEvents.Set(uint64(value))
}

func (ob *queueObserver) MaxBytes(value int) {
	ob.maxBytes.Set(uint64(value))
}

func (ob *queueObserver) Restore(eventCount int, byteCount int) {
	ob.filledEvents.Set(uint64(eventCount))
	ob.filledBytes.Set(uint64(byteCount))
	ob.updateFilledPct()
}

func (ob *queueObserver) AddEvent(byteCount int) {
	ob.addedEvents.Inc()
	ob.addedBytes.Add(uint64(byteCount))

	ob.filledEvents.Inc()
	ob.filledBytes.Add(uint64(byteCount))
	ob.updateFilledPct()
}

func (ob *queueObserver) ConsumeEvents(eventCount int, byteCount int) {
	ob.consumedEvents.Add(uint64(eventCount))
	ob.consumedBytes.Add(uint64(byteCount))
}

func (ob *queueObserver) RemoveEvents(eventCount int, byteCount int) {
	ob.removedEvents.Add(uint64(eventCount))
	ob.acked.Add(uint64(eventCount))
	ob.removedBytes.Add(uint64(byteCount))

	ob.filledEvents.Sub(uint64(eventCount))
	ob.filledBytes.Sub(uint64(byteCount))
	ob.updateFilledPct()
}

func (ob *queueObserver) updateFilledPct() {
	if maxBytes := ob.maxBytes.Get(); maxBytes > 0 {
		ob.filledPct.Set(float64(ob.filledBytes.Get()) / float64(maxBytes))
	} else {
		ob.filledPct.Set(float64(ob.filledEvents.Get()) / float64(ob.maxEvents.Get()))
	}
}

func (nilObserver) MaxEvents(_ int)            {}
func (nilObserver) MaxBytes(_ int)             {}
func (nilObserver) Restore(_ int, _ int)       {}
func (nilObserver) AddEvent(_ int)             {}
func (nilObserver) ConsumeEvents(_ int, _ int) {}
func (nilObserver) RemoveEvents(_ int, _ int)  {}
