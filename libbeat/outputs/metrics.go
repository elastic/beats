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

package outputs

import (
	"time"

	"github.com/rcrowley/go-metrics"

	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

// Stats implements the Observer interface, for collecting metrics on common
// outputs events.
type Stats struct {
	//
	// Output event stats
	//

	// Number of calls to the output's Publish function
	eventsBatches *monitoring.Uint

	// Number of events sent to the output's Publish function.
	eventsTotal *monitoring.Uint

	// Number of events accepted by the output's receiver.
	eventsACKed *monitoring.Uint

	// Number of events that reported a retryable error from the output's
	// receiver.
	eventsFailed *monitoring.Uint

	// Number of events that were dropped due to a non-retryable error.
	eventsDropped *monitoring.Uint

	// Number of events rejected by the output's receiver for being duplicates.
	eventsDuplicates *monitoring.Uint

	// (Gauge) Number of events that have been sent to the output's Publish
	// call but have not yet been ACKed,
	eventsActive *monitoring.Uint // (gauge) events sent and waiting for ACK/fail from output

	// Number of events that failed due to a "429 too many requests" error.
	// These events are also included in eventsFailed.
	eventsTooMany *monitoring.Uint

	// Output batch stats

	// Number of times a batch was split for being too large
	batchesSplit *monitoring.Uint

	//
	// Output network connection stats
	//
	writeBytes  *monitoring.Uint // total amount of bytes written by output
	writeErrors *monitoring.Uint // total number of errors on write

	readBytes  *monitoring.Uint // total amount of bytes read
	readErrors *monitoring.Uint // total number of errors while waiting for response on output

	sendLatencyMillis metrics.Sample
}

// NewStats creates a new Stats instance using a backing monitoring registry.
// This function will create and register a number of metrics with the registry passed.
// The registry must not be null.
func NewStats(reg *monitoring.Registry) *Stats {
	obj := &Stats{
		eventsBatches:    monitoring.NewUint(reg, "events.batches"),
		eventsTotal:      monitoring.NewUint(reg, "events.total"),
		eventsACKed:      monitoring.NewUint(reg, "events.acked"),
		eventsFailed:     monitoring.NewUint(reg, "events.failed"),
		eventsDropped:    monitoring.NewUint(reg, "events.dropped"),
		eventsDuplicates: monitoring.NewUint(reg, "events.duplicates"),
		eventsActive:     monitoring.NewUint(reg, "events.active"),
		eventsTooMany:    monitoring.NewUint(reg, "events.toomany"),

		batchesSplit: monitoring.NewUint(reg, "batches.split"),

		writeBytes:  monitoring.NewUint(reg, "write.bytes"),
		writeErrors: monitoring.NewUint(reg, "write.errors"),

		readBytes:  monitoring.NewUint(reg, "read.bytes"),
		readErrors: monitoring.NewUint(reg, "read.errors"),

		sendLatencyMillis: metrics.NewUniformSample(1024),
	}
	_ = adapter.NewGoMetrics(reg, "write.latency", adapter.Accept).Register("histogram", metrics.NewHistogram(obj.sendLatencyMillis))
	return obj
}

// NewBatch updates active batch and event metrics.
func (s *Stats) NewBatch(n int) {
	if s != nil {
		s.eventsBatches.Inc()
		s.eventsTotal.Add(uint64(n))
		s.eventsActive.Add(uint64(n))
	}
}

func (s *Stats) ReportLatency(time time.Duration) {
	s.sendLatencyMillis.Update(time.Milliseconds())
}

// AckedEvents updates active and acked event metrics.
func (s *Stats) AckedEvents(n int) {
	if s != nil {
		s.eventsACKed.Add(uint64(n))
		s.eventsActive.Sub(uint64(n))
	}
}

// RetryableErrors updates active and failed event metrics.
func (s *Stats) RetryableErrors(n int) {
	if s != nil {
		s.eventsFailed.Add(uint64(n))
		s.eventsActive.Sub(uint64(n))
	}
}

// DuplicateEvents updates the active and duplicate event metrics.
func (s *Stats) DuplicateEvents(n int) {
	if s != nil {
		s.eventsDuplicates.Add(uint64(n))
		s.eventsActive.Sub(uint64(n))
	}
}

// PermanentErrors updates total number of event drops as reported by the output.
// Outputs will only report dropped events on fatal errors which lead to the
// event not being publishable. For example encoding errors or total event size
// being bigger then maximum supported event size.
func (s *Stats) PermanentErrors(n int) {
	// number of dropped events (e.g. encoding failures)
	if s != nil {
		s.eventsActive.Sub(uint64(n))
		s.eventsDropped.Add(uint64(n))
	}
}

// CancelledEvents updates the active event metrics.
func (s *Stats) CancelledEvents(n int) {
	if s != nil {
		s.eventsActive.Sub(uint64(n))
	}
}

func (s *Stats) BatchSplit() {
	if s != nil {
		s.batchesSplit.Inc()
	}
}

// ErrTooMany updates the number of Too Many Requests responses reported by the output.
func (s *Stats) ErrTooMany(n int) {
	if s != nil {
		s.eventsTooMany.Add(uint64(n))
	}
}

// WriteError increases the write I/O error metrics.
func (s *Stats) WriteError(err error) {
	if s != nil {
		s.writeErrors.Inc()
	}
}

// WriteBytes updates the total number of bytes written/send by an output.
func (s *Stats) WriteBytes(n int) {
	if s != nil {
		s.writeBytes.Add(uint64(n))
	}
}

// ReadError increases the read I/O error metrics.
func (s *Stats) ReadError(err error) {
	if s != nil {
		s.readErrors.Inc()
	}
}

// ReadBytes updates the total number of bytes read/received by an output.
func (s *Stats) ReadBytes(n int) {
	if s != nil {
		s.readBytes.Add(uint64(n))
	}
}
