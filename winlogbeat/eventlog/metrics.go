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

package eventlog

import (
	"expvar"
	"strconv"
	"syscall"
	"time"

	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

var (
	// dropReasons contains counters for the number of dropped events for each
	// reason.
	dropReasons = expvar.NewMap("drop_reasons")

	// readErrors contains counters for the read error types that occur.
	readErrors = expvar.NewMap("read_errors")
)

// incrementMetric increments a value in the specified expvar.Map. The key
// should be a windows syscall.Errno or a string. Any other types will be
// reported under the "other" key.
func incrementMetric(v *expvar.Map, key interface{}) {
	switch t := key.(type) {
	default:
		v.Add("other", 1)
	case string:
		v.Add(t, 1)
	case syscall.Errno:
		v.Add(strconv.Itoa(int(t)), 1)
	}
}

// inputMetrics handles event log metric reporting.
type inputMetrics struct {
	unregister func()

	lastBatch time.Time

	name        *monitoring.String // name of the provider being read
	events      *monitoring.Uint   // total number of events received
	dropped     *monitoring.Uint   // total number of discarded events
	errors      *monitoring.Uint   // total number of errors
	batchSize   metrics.Sample     // histogram of the number of events in each non-zero batch
	sourceLag   metrics.Sample     // histogram of the difference between timestamped event's creation and reading
	batchPeriod metrics.Sample     // histogram of the elapsed time between non-zero batch reads
}

// newInputMetrics returns an input metric for windows event logs. If id is empty
// a nil inputMetric is returned.
func newInputMetrics(name, id string) *inputMetrics {
	if id == "" {
		return nil
	}
	reg, unreg := inputmon.NewInputRegistry("winlog", id, nil)
	out := &inputMetrics{
		unregister:  unreg,
		name:        monitoring.NewString(reg, "provider"),
		events:      monitoring.NewUint(reg, "received_events_total"),
		dropped:     monitoring.NewUint(reg, "discarded_events_total"),
		errors:      monitoring.NewUint(reg, "errors_total"),
		batchSize:   metrics.NewUniformSample(1024),
		sourceLag:   metrics.NewUniformSample(1024),
		batchPeriod: metrics.NewUniformSample(1024),
	}
	out.name.Set(name)
	_ = adapter.NewGoMetrics(reg, "received_events_count", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.batchSize))
	_ = adapter.NewGoMetrics(reg, "source_lag_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.sourceLag))
	_ = adapter.NewGoMetrics(reg, "batch_read_period", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.batchPeriod))

	return out
}

// log logs metric for the given batch.
func (m *inputMetrics) log(batch []Record) {
	if m == nil {
		return
	}
	if len(batch) == 0 {
		return
	}

	now := time.Now()
	if !m.lastBatch.IsZero() {
		m.batchPeriod.Update(now.Sub(m.lastBatch).Nanoseconds())
	}
	m.lastBatch = now

	m.events.Add(uint64(len(batch)))
	m.batchSize.Update(int64(len(batch)))
	for _, r := range batch {
		m.sourceLag.Update(now.Sub(r.TimeCreated.SystemTime).Nanoseconds())
	}
}

// logError logs error metrics. Nil errors do not increment the error
// count but the err value is currently otherwise not used. It is included
// to allow easier extension of the metrics to include error stratification.
func (m *inputMetrics) logError(err error) {
	if m == nil {
		return
	}
	if err == nil {
		return
	}
	m.errors.Inc()
}

// logDropped logs dropped event metrics. Nil errors *do* increment the dropped
// count; the value is currently otherwise not used, but is included to allow
// easier extension of the metrics to include error stratification.
func (m *inputMetrics) logDropped(_ error) {
	if m == nil {
		return
	}
	m.dropped.Inc()
}

func (m *inputMetrics) close() {
	if m == nil {
		return
	}
	m.unregister()
}
