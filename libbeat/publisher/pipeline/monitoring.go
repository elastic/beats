package pipeline

import "github.com/elastic/beats/libbeat/monitoring"

type observer interface {
	pipelineObserver
	clientObserver
	queueObserver
	outputObserver

	cleanup()
}

type pipelineObserver interface {
	clientConnected()
	clientClosing()
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
}

type outputObserver interface {
	updateOutputGroup()
	eventsFailed(int)
	eventsDropped(int)
	eventsRetry(int)
	outBatchSend(int)
	outBatchACKed(int)
}

// metricsObserver is used by many component in the publisher pipeline, to report
// internal events. The oberserver can call registered global event handlers or
// updated shared counters/metrics for reporting.
// All events required for reporting events/metrics on the pipeline-global level
// are defined by observer. The components are only allowed to serve localized
// event-handlers only (e.g. the client centric events callbacks)
type metricsObserver struct {
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

func newMetricsObserver(metrics *monitoring.Registry) *metricsObserver {
	reg := metrics.GetRegistry("pipeline")
	if reg == nil {
		reg = metrics.NewRegistry("pipeline")
	}

	return &metricsObserver{
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

func (o *metricsObserver) cleanup() {
	if o.metrics != nil {
		o.metrics.Remove("pipeline") // drop all metrics from registry
	}
}

//
// client connects/disconnects
//

// (pipeline) pipeline did finish creating a new client instance
func (o *metricsObserver) clientConnected() { o.clients.Inc() }

// (client) close being called on client
func (o *metricsObserver) clientClosing() {}

// (client) client finished processing close
func (o *metricsObserver) clientClosed() { o.clients.Dec() }

//
// client publish events
//

// (client) client is trying to publish a new event
func (o *metricsObserver) newEvent() {
	o.events.Inc()
	o.activeEvents.Inc()
}

// (client) event is filtered out (on purpose or failed)
func (o *metricsObserver) filteredEvent() {
	o.filtered.Inc()
	o.activeEvents.Dec()
}

// (client) managed to push an event into the publisher pipeline
func (o *metricsObserver) publishedEvent() {
	o.published.Inc()
}

// (client) client closing down or DropIfFull is set
func (o *metricsObserver) failedPublishEvent() {
	o.failed.Inc()
	o.activeEvents.Dec()
}

//
// queue events
//

// (queue) number of events ACKed by the queue/broker in use
func (o *metricsObserver) queueACKed(n int) {
	o.ackedQueue.Add(uint64(n))
	o.activeEvents.Sub(uint64(n))
}

//
// pipeline output events
//

// (controller) new output group is about to be loaded
func (o *metricsObserver) updateOutputGroup() {}

// (retryer) new failed batch has been received
func (o *metricsObserver) eventsFailed(int) {}

// (retryer) number of events dropped by retryer
func (o *metricsObserver) eventsDropped(n int) {
	o.dropped.Add(uint64(n))
}

// (retryer) number of events pushed to the output worker queue
func (o *metricsObserver) eventsRetry(n int) {
	o.retry.Add(uint64(n))
}

// (output) number of events to be forwarded to the output client
func (o *metricsObserver) outBatchSend(int) {}

// (output) number of events acked by the output batch
func (o *metricsObserver) outBatchACKed(int) {}

type emptyObserver struct{}

var nilObserver observer = (*emptyObserver)(nil)

func (*emptyObserver) cleanup()            {}
func (*emptyObserver) clientConnected()    {}
func (*emptyObserver) clientClosing()      {}
func (*emptyObserver) clientClosed()       {}
func (*emptyObserver) newEvent()           {}
func (*emptyObserver) filteredEvent()      {}
func (*emptyObserver) publishedEvent()     {}
func (*emptyObserver) failedPublishEvent() {}
func (*emptyObserver) queueACKed(n int)    {}
func (*emptyObserver) updateOutputGroup()  {}
func (*emptyObserver) eventsFailed(int)    {}
func (*emptyObserver) eventsDropped(int)   {}
func (*emptyObserver) eventsRetry(int)     {}
func (*emptyObserver) outBatchSend(int)    {}
func (*emptyObserver) outBatchACKed(int)   {}
