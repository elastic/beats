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

package pipeline

import (
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type observer interface {
	pipelineObserver
	clientObserver
	retryObserver

	cleanup()
}

type pipelineObserver interface {
	// A new client connected to the pipeline via (*Pipeline).ConnectWith.
	clientConnected()
	// An open pipeline client received a Close() call.
	clientClosed()
}

type clientObserver interface {
	// The client received a Publish call
	newEvent(inputMetrics)
	// An event was filtered by processors before being published
	filteredEvent(inputMetrics)
	// An event was published to the queue
	publishedEvent(inputMetrics)
	// An event was rejected by the queue
	failedPublishEvent(inputMetrics)
	eventsACKed(count int)
}

type retryObserver interface {
	// Events encountered too many errors and were permanently dropped.
	eventsDropped(int)
	// Events were sent back to an output worker after an earlier failure.
	eventsRetry(int)
}

// metricsObserver is used by many components in the publisher pipeline, to report
// internal events. The observer can call registered global event handlers or
// updated shared counters/metrics for reporting.
// All events required for reporting events/metrics on the pipeline-global level
// are defined by observer. The components are only allowed to serve localized
// event-handlers only (e.g. the client centric events callbacks)
type metricsObserver struct {
	metrics *monitoring.Registry
	vars    metricsObserverVars
}

type inputMetrics struct {
	inputEventsTotal,
	inputEventsFiltered,
	inputEventsPublished *monitoring.Uint
}

type metricsObserverVars struct {
	// clients metrics
	clients *monitoring.Uint

	// eventsTotal publish/dropped stats
	eventsTotal, eventsFiltered, eventsPublished, eventsFailed *monitoring.Uint

	eventsDropped, eventsRetry *monitoring.Uint // (retryer) drop/retry counters
	activeEvents               *monitoring.Uint
}

func newMetricsObserver(metrics *monitoring.Registry) *metricsObserver {
	reg := metrics.GetRegistry("pipeline")
	if reg == nil {
		reg = metrics.NewRegistry("pipeline")
	}

	return &metricsObserver{
		metrics: metrics,

		vars: metricsObserverVars{
			// (Gauge) clients measures the number of open pipeline clients.
			clients: monitoring.NewUint(reg, "clients"),

			// events.total counts all created events.
			eventsTotal: monitoring.NewUint(reg, "events.total"),

			// (Gauge) events.active measures events that have been created, but have
			// not yet been failed, filtered, or acked/dropped.
			activeEvents: monitoring.NewUint(reg, "events.active"),

			// events.filtered counts events that were filtered by processors before
			// being sent to the queue.
			eventsFiltered: monitoring.NewUint(reg, "events.filtered"),

			// events.failed counts events that were rejected by the queue, or that
			// were sent via an already-closed pipeline client.
			eventsFailed: monitoring.NewUint(reg, "events.failed"),

			// events.published counts events that were accepted by the queue.
			eventsPublished: monitoring.NewUint(reg, "events.published"),

			// events.retry counts events that an output worker sent back to be
			// retried.
			eventsRetry: monitoring.NewUint(reg, "events.retry"),

			// events.dropped counts events that were dropped because errors from
			// the output workers exceeded the configured maximum retry count.
			eventsDropped: monitoring.NewUint(reg, "events.dropped"),
		},
	}
}

func (o *metricsObserver) cleanup() {
	if o.metrics != nil {
		o.metrics.Remove("pipeline") // drop all metrics from registry
	}
}

//
// client connects/disconnects
//

// (pipeline) pipeline did finish creating a new client instance
func (o *metricsObserver) clientConnected() { o.vars.clients.Inc() }

// (client) client finished processing close
func (o *metricsObserver) clientClosed() { o.vars.clients.Dec() }

//
// client publish events
//

// (client) client is trying to publish a new event
func (o *metricsObserver) newEvent(inputMetrics inputMetrics) {
	o.vars.eventsTotal.Inc()
	o.vars.activeEvents.Inc()
	inputMetrics.inputEventsTotal.Inc()
}

// (client) event is filtered out (on purpose or failed)
func (o *metricsObserver) filteredEvent(inputMetrics inputMetrics) {
	o.vars.eventsFiltered.Inc()
	o.vars.activeEvents.Dec()
	inputMetrics.inputEventsFiltered.Inc()
}

// (client) managed to push an event into the publisher pipeline
func (o *metricsObserver) publishedEvent(inputMetrics inputMetrics) {
	o.vars.eventsPublished.Inc()
	inputMetrics.inputEventsPublished.Inc()
}

// (client) number of ACKed events from this client
func (o *metricsObserver) eventsACKed(n int) {
	o.vars.activeEvents.Sub(uint64(n))
}

// (client) client closing down or DropIfFull is set
func (o *metricsObserver) failedPublishEvent(inputMetrics inputMetrics) {
	o.vars.eventsFailed.Inc()
	o.vars.activeEvents.Dec()
}

//
// pipeline output events
//

// (retryer) number of events dropped by retryer
func (o *metricsObserver) eventsDropped(n int) {
	o.vars.eventsDropped.Add(uint64(n))
}

// (retryer) number of events pushed to the output worker queue
func (o *metricsObserver) eventsRetry(n int) {
	o.vars.eventsRetry.Add(uint64(n))
}

type emptyObserver struct{}

var nilObserver observer = (*emptyObserver)(nil)

func (*emptyObserver) cleanup()                        {}
func (*emptyObserver) clientConnected()                {}
func (*emptyObserver) clientClosed()                   {}
func (*emptyObserver) newEvent(inputMetrics)           {}
func (*emptyObserver) filteredEvent(inputMetrics)      {}
func (*emptyObserver) publishedEvent(inputMetrics)     {}
func (*emptyObserver) failedPublishEvent(inputMetrics) {}
func (*emptyObserver) eventsACKed(n int)               {}
func (*emptyObserver) eventsDropped(int)               {}
func (*emptyObserver) eventsRetry(int)                 {}
