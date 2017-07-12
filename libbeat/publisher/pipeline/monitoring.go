package pipeline

import "github.com/elastic/beats/libbeat/monitoring"

// observer is used by many component in the publisher pipeline, to report
// internal events. The oberserver can call registered global event handlers or
// updated shared counters/metrics for reporting.
// All events required for reporting events/metrics on the pipeline-global level
// are defined by observer. The components are only allowed to serve localized
// event-handlers only (e.g. the client centric events callbacks)
type observer struct {
	metrics *monitoring.Registry

	// clients metrics
	clients *monitoring.Uint

	// events publish/dropped stats
	events, filtered, published, failed *monitoring.Uint
	dropped, retry                      *monitoring.Uint // (retryer) drop/retry counters
	activeEvents                        *monitoring.Uint

	// queue metrics
	ackedQueue *monitoring.Uint
}

func (o *observer) init(metrics *monitoring.Registry) {
	o.metrics = metrics
	reg := metrics.GetRegistry("pipeline")
	if reg == nil {
		reg = metrics.NewRegistry("pipeline")
	}

	*o = observer{
		metrics: metrics,
		clients: monitoring.NewUint(reg, "clients"),

		events:    monitoring.NewUint(reg, "events.total"),
		filtered:  monitoring.NewUint(reg, "events.filtered"),
		published: monitoring.NewUint(reg, "events.published"),
		failed:    monitoring.NewUint(reg, "events.failed"),
		dropped:   monitoring.NewUint(reg, "events.dropped"),
		retry:     monitoring.NewUint(reg, "events.retry"),

		ackedQueue: monitoring.NewUint(reg, "queue.acked"),

		activeEvents: monitoring.NewUint(reg, "events.active"),
	}
}

func (o *observer) cleanup() {
	o.metrics.Remove("pipeline") // drop all metrics from registry
}

//
// client connects/disconnects
//

// (pipeline) pipeline did finish creating a new client instance
func (o *observer) clientConnected() { o.clients.Inc() }

// (client) close being called on client
func (o *observer) clientClosing() {}

// (client) client finished processing close
func (o *observer) clientClosed() { o.clients.Dec() }

//
// client publish events
//

// (client) client is trying to publish a new event
func (o *observer) newEvent() {
	o.events.Inc()
	o.activeEvents.Inc()
}

// (client) event is filtered out (on purpose or failed)
func (o *observer) filteredEvent() {
	o.filtered.Inc()
	o.activeEvents.Dec()
}

// (client) managed to push an event into the publisher pipeline
func (o *observer) publishedEvent() {
	o.published.Inc()
}

// (client) client closing down or DropIfFull is set
func (o *observer) failedPublishEvent() {
	o.failed.Inc()
	o.activeEvents.Dec()
}

//
// queue events
//

// (queue) number of events ACKed by the queue/broker in use
func (o *observer) queueACKed(n int) {
	o.ackedQueue.Add(uint64(n))
	o.activeEvents.Sub(uint64(n))
}

//
// pipeline output events
//

// (controller) new output group is about to be loaded
func (o *observer) updateOutputGroup() {}

// (retryer) new failed batch has been received
func (o *observer) eventsFailed(int) {}

// (retryer) number of events dropped by retryer
func (o *observer) eventsDropped(n int) {
	o.dropped.Add(uint64(n))
}

// (retryer) number of events pushed to the output worker queue
func (o *observer) eventsRetry(n int) {
	o.retry.Add(uint64(n))
}

// (output) number of events to be forwarded to the output client
func (o *observer) outBatchSend(int) {}

// (output) number of events acked by the output batch
func (o *observer) outBatchACKed(int) {}
