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

import "github.com/elastic/beats/v7/libbeat/monitoring"

type observer interface {
	pipelineObserver
	clientObserver
	queueObserver
	outputObserver

	cleanup()
}

type pipelineObserver interface {
	clientConnected()
	clientClosed()
}

type clientObserver interface {
	newEvent()
	filteredEvent()
	publishedEvent()
	failedPublishEvent()
}

type queueObserver interface {
	queueACKed(n int)
	queueMaxEvents(n int)
}

type outputObserver interface {
	eventsDropped(int)
	eventsRetry(int)
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

	// events publish/dropped stats
	events, filtered, published, failed *monitoring.Uint
	dropped, retry                      *monitoring.Uint // (retryer) drop/retry counters
	activeEvents                        *monitoring.Uint

	// queue metrics
	queueACKed     *monitoring.Uint
	queueMaxEvents *monitoring.Uint
}

func newMetricsObserver(metrics *monitoring.Registry) *metricsObserver {
	reg := metrics.GetRegistry("pipeline")
	if reg == nil {
		reg = metrics.NewRegistry("pipeline")
	}

	return &metricsObserver{
		metrics: metrics,
		vars: metricsObserverVars{
			clients: monitoring.NewUint(reg, "clients"),

			events:    monitoring.NewUint(reg, "events.total"),
			filtered:  monitoring.NewUint(reg, "events.filtered"),
			published: monitoring.NewUint(reg, "events.published"),
			failed:    monitoring.NewUint(reg, "events.failed"),
			dropped:   monitoring.NewUint(reg, "events.dropped"),
			retry:     monitoring.NewUint(reg, "events.retry"),

			queueACKed:     monitoring.NewUint(reg, "queue.acked"),
			queueMaxEvents: monitoring.NewUint(reg, "queue.max_events"),

			activeEvents: monitoring.NewUint(reg, "events.active"),
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
	o.vars.events.Inc()
	o.vars.activeEvents.Inc()
}

// (client) event is filtered out (on purpose or failed)
func (o *metricsObserver) filteredEvent() {
	o.vars.filtered.Inc()
	o.vars.activeEvents.Dec()
}

// (client) managed to push an event into the publisher pipeline
func (o *metricsObserver) publishedEvent() {
	o.vars.published.Inc()
}

// (client) client closing down or DropIfFull is set
func (o *metricsObserver) failedPublishEvent() {
	o.vars.failed.Inc()
	o.vars.activeEvents.Dec()
}

//
// queue events
//

// (queue) number of events ACKed by the queue/broker in use
func (o *metricsObserver) queueACKed(n int) {
	o.vars.queueACKed.Add(uint64(n))
	o.vars.activeEvents.Sub(uint64(n))
}

// (queue) maximum queue event capacity
func (o *metricsObserver) queueMaxEvents(n int) {
	o.vars.queueMaxEvents.Set(uint64(n))
}

//
// pipeline output events
//

// (retryer) number of events dropped by retryer
func (o *metricsObserver) eventsDropped(n int) {
	o.vars.dropped.Add(uint64(n))
}

// (retryer) number of events pushed to the output worker queue
func (o *metricsObserver) eventsRetry(n int) {
	o.vars.retry.Add(uint64(n))
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
