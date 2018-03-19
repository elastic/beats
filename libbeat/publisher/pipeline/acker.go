package pipeline

import (
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common/atomic"
)

// acker is used to account for published and non-published events to be ACKed
// to the beats client.
// All pipeline and client ACK handling support is provided by acker instances.
type acker interface {
	close()
	addEvent(event beat.Event, published bool) bool
	ackEvents(int)
}

// emptyACK ignores any ACK signals and events.
type emptyACK struct{}

var nilACKer acker = (*emptyACK)(nil)

func (*emptyACK) close()                             {}
func (*emptyACK) addEvent(_ beat.Event, _ bool) bool { return true }
func (*emptyACK) ackEvents(_ int)                    {}

type ackerFn struct {
	Close     func()
	AddEvent  func(beat.Event, bool) bool
	AckEvents func(int)
}

func (a *ackerFn) close()                             { a.Close() }
func (a *ackerFn) addEvent(e beat.Event, b bool) bool { return a.AddEvent(e, b) }
func (a *ackerFn) ackEvents(n int)                    { a.AckEvents(n) }

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

func (a *countACK) close()                             {}
func (a *countACK) addEvent(_ beat.Event, _ bool) bool { return true }
func (a *countACK) ackEvents(n int) {
	if a.pipeline.ackActive.Load() {
		a.fn(n, n)
	}
}

// gapCountACK returns event ACKs to the producer, taking account for dropped events.
// Events being dropped by processors will always be ACKed with the last batch ACKed
// by the broker. This way clients waiting for ACKs can expect all processed
// events being always ACKed.
type gapCountACK struct {
	pipeline *Pipeline

	fn func(total int, acked int)

	done chan struct{}

	drop chan struct{}
	acks chan int

	events atomic.Uint32
	lst    gapList
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
		pipeline: pipeline,
		fn:       fn,
		done:     make(chan struct{}),
		drop:     make(chan struct{}),
		acks:     make(chan int, 1),
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

	acks, drop := a.acks, a.drop
	closing := false

	for {
		select {
		case <-a.done:
			closing = true
			a.done = nil

		case <-a.pipeline.ackDone:
			return

		case n := <-acks:
			empty := a.handleACK(n)
			if empty && closing && a.events.Load() == 0 {
				// stop worker, iff all events accounted for have been ACKed
				return
			}

		case <-drop:
			// TODO: accumulate multiple drop events + flush count with timer
			a.fn(1, 0)
		}
	}
}

func (a *gapCountACK) handleACK(n int) bool {
	// collect items and compute total count from gapList

	var (
		total    = 0
		acked    = n
		emptyLst bool
	)

	for n > 0 {
		if emptyLst {
			panic("too many events acked")
		}

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
			current.send -= n
			total += n
			n = 0
		} else {
			total += current.send + current.dropped
			n -= current.send
			current.dropped = 0
			current.send = 0
		}
		current.Unlock()
	}

	a.events.Sub(uint32(total))
	a.fn(total, acked)
	return emptyLst
}

func (a *gapCountACK) close() {
	// close client only, pipeline itself still can handle pending ACKs
	close(a.done)
}

func (a *gapCountACK) addEvent(_ beat.Event, published bool) bool {
	// if gapList is empty and event is being dropped, forward drop event to ack
	// loop worker:

	a.events.Inc()
	if !published {
		a.addDropEvent()
	} else {
		a.addPublishedEvent()
	}

	return true
}

func (a *gapCountACK) addDropEvent() {
	a.lst.Lock()

	current := a.lst.tail
	current.Lock()

	if current.send == 0 && current.next == nil {
		// send can only be 0 if no no events/gaps present yet
		if a.lst.head != a.lst.tail {
			panic("gap list expected to be empty")
		}

		current.Unlock()
		a.lst.Unlock()

		a.drop <- struct{}{}
	} else {
		a.lst.Unlock()

		current.dropped++
		current.Unlock()
	}
}

func (a *gapCountACK) addPublishedEvent() {
	// event is publisher -> add a new gap list entry if gap is present in current
	// gapInfo

	a.lst.Lock()

	current := a.lst.tail
	current.Lock()

	if current.dropped > 0 {
		tmp := &gapInfo{}
		a.lst.tail.next = tmp
		a.lst.tail = tmp

		current.Unlock()
		tmp.Lock()
		current = tmp
	}

	a.lst.Unlock()

	current.send++
	current.Unlock()
}

func (a *gapCountACK) ackEvents(n int) {
	select {
	case <-a.pipeline.ackDone: // pipeline is closing down -> ignore event
		a.acks = nil
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

func (a *boundGapCountACK) addEvent(event beat.Event, published bool) bool {
	a.sema.inc()
	return a.acker.addEvent(event, published)
}

func (a *boundGapCountACK) ackEvents(n int) { a.acker.ackEvents(n) }
func (a *boundGapCountACK) onACK(total, acked int) {
	a.sema.release(total)
	a.fn(total, acked)
}

// eventDataACK reports all dropped and ACKed events private fields.
// An instance of eventDataACK requires a counting ACKer (boundGapCountACK or countACK),
// for accounting for potentially dropped events.
type eventDataACK struct {
	mutex sync.Mutex

	acker    acker
	pipeline *Pipeline

	// TODO: replace with more efficient dynamic sized ring-buffer?
	data []interface{}
	fn   func(data []interface{}, acked int)
}

func newEventACK(
	pipeline *Pipeline,
	canDrop bool,
	sema *sema,
	fn func([]interface{}, int),
) *eventDataACK {
	a := &eventDataACK{pipeline: pipeline, fn: fn}
	a.acker = makeCountACK(pipeline, canDrop, sema, a.onACK)

	return a
}

func makeCountACK(pipeline *Pipeline, canDrop bool, sema *sema, fn func(int, int)) acker {
	if canDrop {
		return newBoundGapCountACK(pipeline, sema, fn)
	}
	return newCountACK(fn)
}

func (a *eventDataACK) close() {
	a.acker.close()
}

func (a *eventDataACK) addEvent(event beat.Event, published bool) bool {
	a.mutex.Lock()
	active := a.pipeline.ackActive.Load()
	if active {
		a.data = append(a.data, event.Private)
	}
	a.mutex.Unlock()

	if active {
		return a.acker.addEvent(event, published)
	}
	return false
}

func (a *eventDataACK) ackEvents(n int) { a.acker.ackEvents(n) }
func (a *eventDataACK) onACK(total, acked int) {
	n := total

	a.mutex.Lock()
	data := a.data[:n]
	a.data = a.data[n:]
	a.mutex.Unlock()

	if len(data) > 0 && a.pipeline.ackActive.Load() {
		a.fn(data, acked)
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

func (a *waitACK) addEvent(event beat.Event, published bool) bool {
	if published {
		a.events.Inc()
	}
	return a.acker.addEvent(event, published)
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
