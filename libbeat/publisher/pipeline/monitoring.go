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
	"math"

	"github.com/elastic/elastic-agent-libs/monitoring"
)

type observer interface {
	pipelineObserver
	clientObserver
	outputObserver

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
	newEvent()
	// An event was filtered by processors before being published
	filteredEvent()
	// An event was published to the queue
	publishedEvent()
	// An event was rejected by the queue
	failedPublishEvent()
}

type outputObserver interface {
	// Events encountered too many errors and were permanently dropped.
	eventsDropped(int)
	// Events were sent back to an output worker after an earlier failure.
	eventsRetry(int)
	// The queue received acknowledgment for events from the output workers.
	// (This may include events already reported via eventsDropped.)
	queueACKed(n int)
	// Report the maximum event count supported by the queue.
	queueMaxEvents(n int)
}

// metricsObserver is used by many component in the publisher pipeline, to report
// internal events. The oberserver can call registered global event handlers or
// updated shared counters/metrics for reporting.
// All events required for reporting events/metrics on the pipeline-global level
// are defined by observer. The components are only allowed to serve localized
// event-handlers only (e.g. the client centric events callbacks)
type metricsObserver struct {
	metrics *monitoring.Registry
	vars    metricsObserverVars
}

type metricsObserverVars struct {
	// clients metrics
	clients *monitoring.Uint

	// eventsTotal publish/dropped stats
	eventsTotal, eventsFiltered, eventsPublished, eventsFailed *monitoring.Uint
	eventsDropped, eventsRetry                                 *monitoring.Uint // (retryer) drop/retry counters
	activeEvents                                               *monitoring.Uint

	// queue metrics
	queueACKed       *monitoring.Uint
	queueMaxEvents   *monitoring.Uint
	percentQueueFull *monitoring.Float
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

			// (Gauge) queue.max_events measures the maximum number of events the
			// queue will accept, or 0 if there is none.
			queueMaxEvents: monitoring.NewUint(reg, "queue.max_events"),

			// queue.acked counts events that have been acknowledged by the output
			// workers. This includes events that were dropped for fatal errors,
			// which are also reported in events.dropped.
			queueACKed: monitoring.NewUint(reg, "queue.acked"),

			// (Gauge) queue.filled.pct.events measures the fraction (from 0 to 1)
			// of the queue's event capacity that is currently filled.
			percentQueueFull: monitoring.NewFloat(reg, "queue.filled.pct.events"),
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
func (o *metricsObserver) newEvent() {
	o.vars.eventsTotal.Inc()
	o.vars.activeEvents.Inc()
	o.setPercentageFull()
}

// setPercentageFull is used interally to set the `queue.full` metric
func (o *metricsObserver) setPercentageFull() {
	maxEvt := o.vars.queueMaxEvents.Get()
	if maxEvt != 0 {
		pct := float64(o.vars.activeEvents.Get()) / float64(maxEvt)
		pctRound := math.Round(pct/0.0005) * 0.0005
		o.vars.percentQueueFull.Set(pctRound)
	}
}

// (client) event is filtered out (on purpose or failed)
func (o *metricsObserver) filteredEvent() {
	o.vars.eventsFiltered.Inc()
	o.vars.activeEvents.Dec()
	o.setPercentageFull()
}

// (client) managed to push an event into the publisher pipeline
func (o *metricsObserver) publishedEvent() {
	o.vars.eventsPublished.Inc()
}

// (client) client closing down or DropIfFull is set
func (o *metricsObserver) failedPublishEvent() {
	o.vars.eventsFailed.Inc()
	o.vars.activeEvents.Dec()
	o.setPercentageFull()
}

//
// queue events
//

// (queue) number of events ACKed by the queue/broker in use
func (o *metricsObserver) queueACKed(n int) {
	o.vars.queueACKed.Add(uint64(n))
	o.vars.activeEvents.Sub(uint64(n))
	o.setPercentageFull()
}

// (queue) maximum queue event capacity
func (o *metricsObserver) queueMaxEvents(n int) {
	o.vars.queueMaxEvents.Set(uint64(n))
	o.setPercentageFull()
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

func (*emptyObserver) cleanup()            {}
func (*emptyObserver) clientConnected()    {}
func (*emptyObserver) clientClosed()       {}
func (*emptyObserver) newEvent()           {}
func (*emptyObserver) filteredEvent()      {}
func (*emptyObserver) publishedEvent()     {}
func (*emptyObserver) failedPublishEvent() {}
func (*emptyObserver) queueACKed(n int)    {}
func (*emptyObserver) queueMaxEvents(int)  {}
func (*emptyObserver) eventsDropped(int)   {}
func (*emptyObserver) eventsRetry(int)     {}
