package mode

import (
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

// LoadBalancerMode balances the sending of events between multiple connections.
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
// Distributing events to workers is subject to timeout. If no worker is available to
// pickup a message for sending, the message will be dropped internally.
type LoadBalancerMode struct {
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

type eventsMessage struct {
	attemptsLeft int
	signaler     outputs.Signaler
	events       []common.MapStr
	event        common.MapStr
}

// NewLoadBalancerMode create a new load balancer connection mode.
func NewLoadBalancerMode(
	clients []ProtocolClient,
	maxAttempts int,
	waitRetry, timeout, maxWaitRetry time.Duration,
) (*LoadBalancerMode, error) {

	// maxAttempts signals infinite retry. Convert to -1, so attempts left and
	// and infinite retry can be more easily distinguished by load balancer
	if maxAttempts == 0 {
		maxAttempts = -1
	}

	m := &LoadBalancerMode{
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
func (m *LoadBalancerMode) Close() error {
	close(m.done)
	m.wg.Wait()
	return nil
}

// PublishEvents forwards events to some load balancing worker.
func (m *LoadBalancerMode) PublishEvents(
	signaler outputs.Signaler,
	opts outputs.Options,
	events []common.MapStr,
) error {
	return m.publishEventsMessage(opts,
		eventsMessage{signaler: signaler, events: events})
}

// PublishEvent forwards the event to some load balancing worker.
func (m *LoadBalancerMode) PublishEvent(
	signaler outputs.Signaler,
	opts outputs.Options,
	event common.MapStr,
) error {
	return m.publishEventsMessage(opts,
		eventsMessage{signaler: signaler, event: event})
}

func (m *LoadBalancerMode) publishEventsMessage(
	opts outputs.Options,
	msg eventsMessage,
) error {
	maxAttempts := m.maxAttempts
	if opts.Guaranteed {
		maxAttempts = -1
	}
	msg.attemptsLeft = maxAttempts

	if ok := m.forwardEvent(m.work, msg); !ok {
		dropping(msg)
	}
	return nil
}

func (m *LoadBalancerMode) start(clients []ProtocolClient) {
	var waitStart sync.WaitGroup
	worker := func(client ProtocolClient) {
		defer func() {
			if client.IsConnected() {
				_ = client.Close()
			}
			m.wg.Done()
		}()

		waitStart.Done()
		m.clientLoop(client)
	}

	for _, client := range clients {
		m.wg.Add(1)
		waitStart.Add(1)
		go worker(client)
	}
	waitStart.Wait()
}

func (m *LoadBalancerMode) clientLoop(client ProtocolClient) {
	debug("load balancer: start client loop")
	defer debug("load balancer: stop client loop")

	backoff := newBackoff(m.done, m.waitRetry, m.maxWaitRetry)

	done := false
	for !done {
		if done = m.connect(client, backoff); !done {
			done = m.sendLoop(client, backoff)
		}
		debug("close client")
		client.Close()
	}
}

func (m *LoadBalancerMode) connect(client ProtocolClient, backoff *backoff) bool {
	for {
		debug("try to (re-)connect client")
		err := client.Connect(m.timeout)
		if !backoff.WaitOnError(err) {
			return true
		}

		if err == nil {
			return false
		}
	}
}

func (m *LoadBalancerMode) sendLoop(client ProtocolClient, backoff *backoff) bool {
	for {
		var msg eventsMessage
		select {
		case <-m.done:
			return true
		case msg = <-m.retries: // receive message from other failed worker
		case msg = <-m.work: // receive message from publisher
		}

		done, err := m.onMessage(backoff, client, msg)
		if done || err != nil {
			return done
		}
	}
}

func (m *LoadBalancerMode) onMessage(
	backoff *backoff,
	client ProtocolClient,
	msg eventsMessage,
) (bool, error) {

	done := false
	if msg.event != nil {
		err := client.PublishEvent(msg.event)
		done = !backoff.WaitOnError(err)
		if err != nil {
			if msg.attemptsLeft > 0 {
				msg.attemptsLeft--
			}
			m.onFail(msg, err)
			return done, err
		}
	} else {
		events := msg.events
		total := len(events)

		for len(events) > 0 {
			var err error

			events, err = client.PublishEvents(events)
			done = !backoff.WaitOnError(err)
			if done && err != nil {
				outputs.SignalFailed(msg.signaler, err)
				return done, err
			}

			if err != nil {
				if msg.attemptsLeft > 0 {
					msg.attemptsLeft--
				}

				// reset attempt count if subset of messages has been processed
				if len(events) < total && msg.attemptsLeft >= 0 {
					debug("reset fails")
					msg.attemptsLeft = m.maxAttempts
				}

				if err != ErrTempBulkFailure {
					// retry non-published subset of events in batch
					msg.events = events
					m.onFail(msg, err)
					return done, err
				}

				if m.maxAttempts > 0 && msg.attemptsLeft == 0 {
					// no more attempts left => drop
					dropping(msg)
					return done, err
				}

				// reset total count for temporary failure loop
				total = len(events)
			}
		}
	}

	outputs.SignalCompleted(msg.signaler)
	return done, nil
}

func (m *LoadBalancerMode) onFail(msg eventsMessage, err error) {

	logp.Info("Error publishing events (retrying): %s", err)

	if !m.forwardEvent(m.retries, msg) {
		dropping(msg)
	}
}

func (m *LoadBalancerMode) forwardEvent(
	ch chan eventsMessage,
	msg eventsMessage,
) bool {
	if msg.attemptsLeft < 0 {
		select {
		case ch <- msg:
			return true
		case <-m.done: // shutdown
			return false
		}
	} else {
		for ; msg.attemptsLeft > 0; msg.attemptsLeft-- {
			select {
			case ch <- msg:
				return true
			case <-m.done: // shutdown
				return false
			case <-time.After(m.timeout):
			}
		}
	}
	return false
}

// dropping is called when a message is dropped. It updates the
// relevant counters and sends a failed signal.
func dropping(msg eventsMessage) {
	debug("messages dropped")
	messagesDropped.Add(1)
	outputs.SignalFailed(msg.signaler, nil)
}
