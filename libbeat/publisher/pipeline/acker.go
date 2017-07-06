package pipeline

import (
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/publisher/beat"
)

// acker is used to account for published and non-published events to be ACKed
// to the beats client.
// All pipeline and client ACK handling support is provided by acker instances.
type acker interface {
	close()
	addEvent(event beat.Event, published bool)
	ackEvents(int)
}

type pipelineAcker interface {
	addEvent(beat.Event, bool)
	ackEvents(int)
}

// emptyACK ignores any ACK signals and events.
type emptyACK struct{}

var nilACKer acker = (*emptyACK)(nil)

func (*emptyACK) close()                        {}
func (*emptyACK) addEvent(_ beat.Event, _ bool) {}
func (*emptyACK) ackEvents(_ int)               {}

type ackerFn struct {
	Close     func()
	AddEvent  func(beat.Event, bool)
	AckEvents func(int)
}

func (a *ackerFn) close()                        { a.Close() }
func (a *ackerFn) addEvent(e beat.Event, b bool) { a.AddEvent(e, b) }
func (a *ackerFn) ackEvents(n int)               { a.AckEvents(n) }

// countACK is used when broker ACK events can be simply forwarded to the
// producers ACKCount callback.
// The countACK is only applicable if no processors are configured.
// ACKs for closed clients will be ignored.
type countACK struct {
	pipeline *Pipeline
	fn       func(total, acked int)
}

func newCountACK(fn func(total, acked int)) *countACK {
	a := &countACK{fn: fn}
	return a
}

func (a *countACK) close()                        {}
func (a *countACK) addEvent(_ beat.Event, _ bool) {}
func (a *countACK) ackEvents(n int) {
	if a.pipeline.ackActive.Load() {
		a.fn(n, n)
	}
}

// gapCountACK returns event ACKs to the producer, taking account for dropped events.
// Events being dropped by processors will always be ACKed with the last batch ACKed
// by the broker. This way clients waiting for ACKs can expect all processed
// events being alwyas ACKed.
type gapCountACK struct {
	pipeline *Pipeline

	fn func(total int, acked int)

	done         chan struct{}
	clientClosed bool

	drop chan struct{}
	acks chan int

	lst gapList
}

type gapList struct {
	sync.Mutex
	head, tail *gapInfo
}

type gapInfo struct {
	sync.Mutex
	next          *gapInfo
	send, dropped int
}

func newGapCountACK(pipeline *Pipeline, fn func(total, acked int)) *gapCountACK {
	a := &gapCountACK{}
	a.init(pipeline, fn)
	return a
}

func (a *gapCountACK) init(pipeline *Pipeline, fn func(int, int)) {
	*a = gapCountACK{
		fn:   fn,
		done: make(chan struct{}),
		drop: make(chan struct{}),
		acks: make(chan int, 1),
	}

	init := &gapInfo{}
	a.lst.head = init
	a.lst.tail = init

	go a.ackLoop()
}

func (a *gapCountACK) ackLoop() {
	// close channels, as no more events should be ACKed:
	// - once pipeline is closed
	// - all events of the closed client have been acked/processed by the pipeline
	defer close(a.drop)
	defer close(a.acks)

	for {
		select {
		case <-a.done:
			a.clientClosed = true
			a.done = nil

		case <-a.pipeline.ackDone:
			return

		case n := <-a.acks:
			empty := a.handleACK(n)
			if empty && a.clientClosed {
				return
			}

		case <-a.drop:
			// TODO: accumulate mulitple drop events + flush count with timer
			a.fn(1, 0)
		}
	}
}

func (a *gapCountACK) handleACK(n int) bool {
	// collect items and compute total count from gapList

	var (
		total    = 0
		emptyLst bool
	)

	for n > 0 {
		a.lst.Lock()
		current := a.lst.head

		current.Lock()
		if n >= current.send {
			nxt := current.next
			emptyLst = nxt == nil
			if !emptyLst {
				// advance list all event in current entry have been send and list as
				// more then 1 gapInfo entry. If only 1 entry is present, list item will be
				// reset and reused
				a.lst.head = nxt
			}
		}

		// hand over lock list-entry, so ACK handler and producer can operate
		// on potentially different list ends
		a.lst.Unlock()

		if n < current.send {
			total += n
			n = 0
			current.send -= n
		} else {
			total += current.send + current.dropped
			n -= current.send
			current.dropped = 0
			current.send = 0
		}
		current.Unlock()
	}

	a.fn(total, n)
	return emptyLst
}

func (a *gapCountACK) close() {
	// close client only, pipeline itself still can handle pending ACKs
	close(a.done)
}

func (a *gapCountACK) addEvent(_ beat.Event, published bool) {
	// if gapList is empty and event is being dropped, forward drop event to ack
	// loop worker:
	if !published {
		a.lst.Lock()
		current := a.lst.tail
		if current.send == 0 {
			a.lst.Unlock()

			// send can only be 0 if no no events/gaps present yet
			if a.lst.head != a.lst.tail {
				panic("gap list expected to be empty")
			}

			a.drop <- struct{}{}
		} else {
			current.Lock()
			a.lst.Unlock()

			current.dropped++
			current.Unlock()
		}

		return
	}

	// event is publisher -> add a new gap list entry if gap is present in current
	// gapInfo

	a.lst.Lock()

	current := a.lst.tail
	if current.dropped > 0 {
		current = &gapInfo{}
		a.lst.tail.next = current
	}

	current.Lock()
	a.lst.Unlock()

	current.send++
	current.Unlock()
}

func (a *gapCountACK) ackEvents(n int) {
	select {
	case <-a.pipeline.ackDone: // pipeline is closing down -> ignore event
	case a.acks <- n: // send ack event to worker
	}
}

// boundGapCountACK guards a gapCountACK instance by bounding the maximum number of
// active events.
// As beats might accumulate state while waiting for ACK, the boundGapCountACK blocks
// if too many events have been filtered out by processors.
type boundGapCountACK struct {
	active bool
	fn     func(total, acked int)

	acker gapCountACK
	sema  *sema
}

func newBoundGapCountACK(
	pipeline *Pipeline,
	sema *sema,
	fn func(total, acked int),
) *boundGapCountACK {
	a := &boundGapCountACK{active: true, sema: sema, fn: fn}
	a.acker.init(pipeline, a.onACK)
	return a
}

func (a *boundGapCountACK) close() {
	a.acker.close()
}

func (a *boundGapCountACK) addEvent(event beat.Event, published bool) {
	a.sema.inc()
	a.acker.addEvent(event, published)
}

func (a *boundGapCountACK) ackEvents(n int) { a.acker.ackEvents(n) }
func (a *boundGapCountACK) onACK(total, acked int) {
	a.sema.release(total)
	a.fn(total, acked)
}

// eventACK reports all dropped and ACKed events.
// An instance of eventACK requires a counting ACKer (boundGapCountACK or countACK),
// for accounting for potentially dropped events.
type eventACK struct {
	mutex  sync.Mutex
	active bool

	acker    acker
	pipeline *Pipeline

	// TODO: replace with more efficient dynamic sized ring-buffer?
	events []beat.Event
	fn     func(events []beat.Event, acked int)
}

func newEventACK(
	pipeline *Pipeline,
	canDrop bool,
	sema *sema,
	fn func([]beat.Event, int),
) *eventACK {
	a := &eventACK{fn: fn}
	a.active = true
	a.acker = makeCountACK(pipeline, canDrop, sema, a.onACK)

	return a
}

func makeCountACK(pipeline *Pipeline, canDrop bool, sema *sema, fn func(int, int)) acker {
	if canDrop {
		return newBoundGapCountACK(pipeline, sema, fn)
	}
	return newCountACK(fn)
}

func (a *eventACK) close() {
	a.mutex.Lock()
	a.active = false
	a.mutex.Unlock()

	a.acker.close()
}

func (a *eventACK) addEvent(event beat.Event, published bool) {
	a.mutex.Lock()
	active := a.active
	if active {
		a.events = append(a.events, event)
	}
	a.mutex.Unlock()

	if active {
		a.acker.addEvent(event, published)
	}
}

func (a *eventACK) ackEvents(n int) { a.acker.ackEvents(n) }
func (a *eventACK) onACK(total, acked int) {
	n := total

	a.mutex.Lock()
	events := a.events[:n]
	a.events = a.events[n:]
	a.mutex.Unlock()

	if len(events) > 0 && a.pipeline.ackActive.Load() {
		a.fn(events, acked)
	}
}

// waitACK keeps track of events being produced and ACKs for events.
// On close waitACK will wait for pending events to be ACKed by the broker.
// The acker continues the closing operation if all events have been published
// or the maximum configured sleep time has been reached.
type waitACK struct {
	acker acker

	signal    chan struct{}
	waitClose time.Duration

	active atomic.Bool

	// number of active events
	events atomic.Uint64
}

func newWaitACK(acker acker, timeout time.Duration) *waitACK {
	return &waitACK{
		acker:     acker,
		signal:    make(chan struct{}, 1),
		waitClose: timeout,
		active:    atomic.MakeBool(true),
	}
}

func (a *waitACK) close() {
	// TODO: wait for events

	a.active.Store(false)
	if a.events.Load() > 0 {
		select {
		case <-a.signal:
		case <-time.After(a.waitClose):
		}
	}

	// close the underlying acker upon exit
	a.acker.close()
}

func (a *waitACK) addEvent(event beat.Event, published bool) {
	if published {
		a.events.Inc()
	}
	a.acker.addEvent(event, published)
}

func (a *waitACK) ackEvents(n int) {
	// return ACK signal to upper layers
	a.acker.ackEvents(n)
	a.releaseEvents(n)
}

func (a *waitACK) releaseEvents(n int) {
	value := a.events.Sub(uint64(n))
	if value != 0 {
		return
	}

	// send done signal, if close is waiting
	if !a.active.Load() {
		a.signal <- struct{}{}
	}

}
