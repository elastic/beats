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

package remote_write

import (
	"net/http"
	"sync"
	"time"
)

// ownerLoopSeams holds optional test-only overrides for a single MetricSet instance.
type ownerLoopSeams struct {
	newTickSource  func(time.Duration) tickSource
	skipHTTPServer bool
}

var (
	ownerLoopTestSeams sync.Map // map[*MetricSet]ownerLoopSeams
	runReadySignals    sync.Map // map[*MetricSet]chan struct{}
)

func setOwnerLoopTestSeams(m *MetricSet, seams ownerLoopSeams) {
	ownerLoopTestSeams.Store(m, seams)
}

func clearOwnerLoopTestSeams(m *MetricSet) {
	ownerLoopTestSeams.Delete(m)
}

func seamsForMetricSet(m *MetricSet) ownerLoopSeams {
	if v, ok := ownerLoopTestSeams.Load(m); ok {
		if seams, ok := v.(ownerLoopSeams); ok {
			return seams
		}
	}
	return ownerLoopSeams{}
}

func tickSourceFactoryFor(m *MetricSet) func(time.Duration) tickSource {
	seams := seamsForMetricSet(m)
	if seams.newTickSource != nil {
		return seams.newTickSource
	}
	return defaultTickSource
}

func shouldStartHTTPServer(m *MetricSet) bool {
	return !seamsForMetricSet(m).skipHTTPServer
}

func registerRunReadySignal(m *MetricSet) chan struct{} {
	ch := make(chan struct{})
	runReadySignals.Store(m, ch)
	return ch
}

func signalRunReady(m *MetricSet) {
	if v, ok := runReadySignals.Load(m); ok {
		if ch, ok := v.(chan struct{}); ok {
			close(ch)
		}
	}
}

func waitRunReady(m *MetricSet) <-chan struct{} {
	if v, ok := runReadySignals.Load(m); ok {
		if ch, ok := v.(chan struct{}); ok {
			return ch
		}
	}
	ch := make(chan struct{})
	close(ch)
	return ch
}

// OwnerLoopTickSource is the flush ticker interface used by MetricSet.Run.
type OwnerLoopTickSource interface {
	C() <-chan time.Time
	Stop()
}

// ConfigureOwnerLoopForTest sets optional owner-loop dependencies for tests.
// skipHTTPServer avoids binding a listener; newTickSource overrides the flush ticker factory.
func ConfigureOwnerLoopForTest(m *MetricSet, newTickSource func(time.Duration) OwnerLoopTickSource, skipHTTPServer bool) {
	var factory func(time.Duration) tickSource
	if newTickSource != nil {
		factory = func(d time.Duration) tickSource {
			return newTickSource(d)
		}
	}
	setOwnerLoopTestSeams(m, ownerLoopSeams{
		newTickSource:  factory,
		skipHTTPServer: skipHTTPServer,
	})
}

// ClearOwnerLoopTestConfiguration removes test seams from m.
func ClearOwnerLoopTestConfiguration(m *MetricSet) {
	clearOwnerLoopTestSeams(m)
}

// WaitOwnerLoopReady returns a channel closed when MetricSet.Run has started the owner loop.
func WaitOwnerLoopReady(m *MetricSet) <-chan struct{} {
	return waitRunReady(m)
}

// HTTPHandler returns the Prometheus remote_write HTTP handler.
func (m *MetricSet) HTTPHandler() http.HandlerFunc {
	return m.handleFunc
}

// SetPromEventsGeneratorForTest replaces the events generator before Run for integration tests.
func (m *MetricSet) SetPromEventsGeneratorForTest(gen RemoteWriteEventsGenerator) {
	m.setPromEventsGenerator(gen)
}

// FlowModeForTest reports the selected mode and allocated intake channels.
func (m *MetricSet) FlowModeForTest() (useOwnerLoop, hasEvents, hasBatches bool) {
	return m.useOwnerLoop, m.events != nil, m.batches != nil
}

// HandlersInFlightForTest reports in-flight HTTP handlers (tests only).
func (m *MetricSet) HandlersInFlightForTest() int32 {
	return m.handlersInFlight.Load()
}
