package lb

import (
	"time"

	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
)

// context distributes event messages among multiple workers. It implements the
// load-balancing strategy itself.
type context struct {
	timeout time.Duration // Send/retry timeout. Every timeout is a failed send attempt

	// maximum number of configured send attempts. If set to 0, publisher will
	// block until event has been successfully published.
	maxAttempts int

	// signaling channel for handling shutdown
	done chan struct{}

	// channels for forwarding work items to workers.
	// The work channel is used by publisher to insert new events
	// into the load balancer. The work channel is synchronous blocking until timeout
	// for one worker available.
	// The retries channel is used to forward failed send attempts to other workers.
	// The retries channel is buffered to mitigate possible deadlocks when all
	// workers become unresponsive.
	work, retries chan eventsMessage
}

type eventsMessage struct {
	worker       int
	attemptsLeft int
	signaler     op.Signaler
	data         []outputs.Data
	datum        outputs.Data
}

func makeContext(nClients, maxAttempts int, timeout time.Duration) context {
	return context{
		timeout:     timeout,
		maxAttempts: maxAttempts,
		done:        make(chan struct{}),
		work:        make(chan eventsMessage),
		retries:     make(chan eventsMessage, nClients*2),
	}
}

func (ctx *context) Close() error {
	debugf("close context")
	close(ctx.done)
	return nil
}

func (ctx *context) pushEvents(msg eventsMessage, guaranteed bool) bool {
	maxAttempts := ctx.maxAttempts
	if guaranteed {
		maxAttempts = -1
	}
	msg.attemptsLeft = maxAttempts
	ok := ctx.forwardEvent(ctx.work, msg)
	if !ok {
		dropping(msg)
	}
	return ok
}

func (ctx *context) pushFailed(msg eventsMessage) bool {
	ok := ctx.forwardEvent(ctx.retries, msg)
	if !ok {
		dropping(msg)
	}
	return ok
}

func (ctx *context) tryPushFailed(msg eventsMessage) bool {
	if msg.attemptsLeft == 0 {
		dropping(msg)
		return true
	}

	select {
	case ctx.retries <- msg:
		return true
	default:
		return false
	}
}

func (ctx *context) forwardEvent(ch chan eventsMessage, msg eventsMessage) bool {
	debugf("forwards msg with attempts=%v", msg.attemptsLeft)

	if msg.attemptsLeft < 0 {
		select {
		case ch <- msg:
			debugf("message forwarded")
			return true
		case <-ctx.done: // shutdown
			debugf("shutting down")
			return false
		}
	} else {
		for ; msg.attemptsLeft > 0; msg.attemptsLeft-- {
			select {
			case ch <- msg:
				debugf("message forwarded")
				return true
			case <-ctx.done: // shutdown
				debugf("shutting down")
				return false
			case <-time.After(ctx.timeout):
				debugf("forward timed out")
			}
		}
	}
	return false
}

func (ctx *context) receive() (eventsMessage, bool) {
	var msg eventsMessage

	select {
	case msg = <-ctx.retries: // receive message from other failed worker
		debugf("events from retries queue")
		return msg, true
	default:
		break
	}

	select {
	case <-ctx.done:
		return msg, false
	case msg = <-ctx.retries: // receive message from other failed worker
		debugf("events from retries queue")
	case msg = <-ctx.work: // receive message from publisher
		debugf("events from worker worker queue")
	}
	return msg, true
}

// dropping is called when a message is dropped. It updates the
// relevant counters and sends a failed signal.
func dropping(msg eventsMessage) {
	debugf("messages dropped")
	mode.Dropped(1)
	op.SigFailed(msg.signaler, nil)
}
