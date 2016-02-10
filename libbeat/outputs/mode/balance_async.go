package mode

import (
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

// AsyncLoadBalancerMode balances the sending of events between multiple connections.
//
// The balancing algorithm is mostly pull-based, with multiple workers trying to pull
// some amount of work from a shared queue. Workers will try to get a new work item
// only if they have a working/active connection. Workers without active connection
// do not participate until a connection has been re-established.
// Due to the pull based nature the algorithm will load-balance events by random
// with workers having less latencies/turn-around times potentially getting more
// work items then other workers with higher latencies. Thusly the algorithm
// dynamically adapts to resource availability of server events are forwarded to.
//
// Workers not participating in the load-balancing will continuously try to reconnect
// to their configured endpoints. Once a new connection has been established,
// these workers will participate in in load-balancing again.
//
// If a connection becomes unavailable, the events are rescheduled for another
// connection to pick up. Rescheduling events is limited to a maximum number of
// send attempts. If events have not been send after maximum number of allowed
// attemps has been passed, they will be dropped.
//
// Like network connections, distributing events to workers is subject to
// timeout. If no worker is available to pickup a message for sending, the message
// will be dropped internally after max_retries. If mode or message requires
// guaranteed send, message is retried infinitely.
type AsyncLoadBalancerMode struct {
	timeout      time.Duration // Send/retry timeout. Every timeout is a failed send attempt
	waitRetry    time.Duration // Duration to wait during re-connection attempts.
	maxWaitRetry time.Duration // Maximum send/retry timeout in backoff case.

	// maximum number of configured send attempts. If set to 0, publisher will
	// block until event has been successfully published.
	maxAttempts int

	// waitGroup + signaling channel for handling shutdown
	wg   sync.WaitGroup
	done chan struct{}

	// channels for forwarding work items to workers.
	// The work channel is used by publisher to insert new events
	// into the load balancer. The work channel is synchronous blocking until timeout
	// for one worker available.
	// The retries channel is used to forward failed send attempts to other workers.
	// The retries channel is buffered to mitigate possible deadlocks when all
	// workers become unresponsive.
	work    chan eventsMessage
	retries chan eventsMessage
}

// NewAsyncLoadBalancerMode create a new load balancer connection mode.
func NewAsyncLoadBalancerMode(
	clients []AsyncProtocolClient,
	maxAttempts int,
	waitRetry, timeout, maxWaitRetry time.Duration,
) (*AsyncLoadBalancerMode, error) {

	debug("configure maxattempts: %v", maxAttempts)

	// maxAttempts signals infinite retry. Convert to -1, so attempts left and
	// and infinite retry can be more easily distinguished by load balancer
	if maxAttempts == 0 {
		maxAttempts = -1
	}

	m := &AsyncLoadBalancerMode{
		timeout:      timeout,
		maxWaitRetry: maxWaitRetry,
		waitRetry:    waitRetry,
		maxAttempts:  maxAttempts,

		work:    make(chan eventsMessage),
		retries: make(chan eventsMessage, len(clients)*2),
		done:    make(chan struct{}),
	}
	m.start(clients)

	return m, nil
}

// Close stops all workers and closes all open connections. In flight events
// are signaled as failed.
func (m *AsyncLoadBalancerMode) Close() error {
	close(m.done)
	m.wg.Wait()
	return nil
}

// PublishEvents forwards events to some load balancing worker.
func (m *AsyncLoadBalancerMode) PublishEvents(
	signaler outputs.Signaler,
	opts outputs.Options,
	events []common.MapStr,
) error {
	return m.publishEventsMessage(opts,
		eventsMessage{signaler: signaler, events: events})
}

// PublishEvent forwards the event to some load balancing worker.
func (m *AsyncLoadBalancerMode) PublishEvent(
	signaler outputs.Signaler,
	opts outputs.Options,
	event common.MapStr,
) error {
	return m.publishEventsMessage(opts,
		eventsMessage{signaler: signaler, event: event})
}

func (m *AsyncLoadBalancerMode) publishEventsMessage(
	opts outputs.Options,
	msg eventsMessage,
) error {
	maxAttempts := m.maxAttempts
	if opts.Guaranteed {
		debug("guaranteed flag is set")
		maxAttempts = -1
	} else {
		debug("guaranteed flag is not set")
	}
	msg.attemptsLeft = maxAttempts
	debug("publish events with attempts=%v", msg.attemptsLeft)

	if ok := m.forwardEvent(m.work, msg); !ok {
		dropping(msg)
	}
	return nil
}

func (m *AsyncLoadBalancerMode) start(clients []AsyncProtocolClient) {
	var waitStart sync.WaitGroup
	worker := func(client AsyncProtocolClient) {
		defer func() {
			if client.IsConnected() {
				_ = client.Close()
			}
			m.wg.Done()
		}()

		waitStart.Done()

		backoff := newBackoff(m.done, m.waitRetry, m.maxWaitRetry)
		for {
			// reconnect loop
			for !client.IsConnected() {
				if err := client.Connect(m.timeout); err == nil {
					break
				}

				if !backoff.Wait() { // done channel closed
					return
				}
			}

			// receive and process messages
			var msg eventsMessage
			select {
			case <-m.done:
				return
			case msg = <-m.retries: // receive message from other failed worker
				debug("events from retries queue")
			case msg = <-m.work: // receive message from publisher
				debug("events from worker worker queue")
			}

			err := m.onMessage(client, msg)
			if !backoff.WaitOnError(err) { // done channel closed
				return
			}
		}
	}

	for _, client := range clients {
		m.wg.Add(1)
		waitStart.Add(1)
		go worker(client)
	}
	waitStart.Wait()
}

func (m *AsyncLoadBalancerMode) onMessage(
	client AsyncProtocolClient,
	msg eventsMessage,
) error {
	var err error
	if msg.event != nil {
		err = client.AsyncPublishEvent(handlePublishEventResult(m, msg), msg.event)
	} else {
		err = client.AsyncPublishEvents(handlePublishEventsResult(m, msg), msg.events)
	}

	if err != nil {
		if msg.attemptsLeft > 0 {
			msg.attemptsLeft--
		}

		// asynchronously retry to insert message (if attempts left), so worker can not
		// deadlock on retries channel if client puts multiple failed outstanding
		// events into the pipeline
		m.onFail(true, msg, err)
	}

	return err
}

func handlePublishEventResult(m *AsyncLoadBalancerMode, msg eventsMessage) func(error) {
	return func(err error) {
		if err != nil {
			if msg.attemptsLeft > 0 {
				msg.attemptsLeft--
			}
			m.onFail(false, msg, err)
		} else {
			outputs.SignalCompleted(msg.signaler)
		}
	}
}

func handlePublishEventsResult(
	m *AsyncLoadBalancerMode,
	msg eventsMessage,
) func([]common.MapStr, error) {
	total := len(msg.events)
	return func(events []common.MapStr, err error) {
		debug("handlePublishEventsResult")

		if err != nil {
			debug("handle publish error: %v", err)

			if msg.attemptsLeft > 0 {
				msg.attemptsLeft--
			}

			// reset attempt count if subset of messages has been processed
			if len(events) < total && msg.attemptsLeft >= 0 {
				msg.attemptsLeft = m.maxAttempts
			}

			if err != ErrTempBulkFailure {
				// retry non-published subset of events in batch
				msg.events = events
				m.onFail(false, msg, err)
				return
			}

			if m.maxAttempts > 0 && msg.attemptsLeft == 0 {
				// no more attempts left => drop
				dropping(msg)
				return
			}

			// retry non-published subset of events in batch
			msg.events = events
			m.onFail(false, msg, err)
			return
		}

		// re-insert non-published events into pipeline
		if len(events) != 0 {
			debug("add non-published events back into pipeline: %v", len(events))
			msg.events = events
			if ok := m.forwardEvent(m.retries, msg); !ok {
				dropping(msg)
			}
			return
		}

		// all events published -> signal success
		debug("async bulk publish success")
		outputs.SignalCompleted(msg.signaler)
	}
}

func (m *AsyncLoadBalancerMode) onFail(async bool, msg eventsMessage, err error) {
	fn := func() {
		logp.Info("Error publishing events (retrying): %s", err)

		if ok := m.forwardEvent(m.retries, msg); !ok {
			dropping(msg)
		}
	}

	if async {
		go fn()
	} else {
		fn()
	}
}

func (m *AsyncLoadBalancerMode) forwardEvent(
	ch chan eventsMessage,
	msg eventsMessage,
) bool {
	debug("forwards msg with attempts=%v", msg.attemptsLeft)

	if msg.attemptsLeft < 0 {
		select {
		case ch <- msg:
			debug("message forwarded")
			return true
		case <-m.done: // shutdown
			debug("shutting down")
			return false
		}
	} else {
		for ; msg.attemptsLeft > 0; msg.attemptsLeft-- {
			select {
			case ch <- msg:
				debug("message forwarded")
				return true
			case <-m.done: // shutdown
				debug("shutting down")
				return false
			case <-time.After(m.timeout):
				debug("forward timed out")
			}
		}
	}
	return false
}
