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

type emptyACK struct{}

// acker ignoring events and any ACK signals
var nilACKer acker = (*emptyACK)(nil)

// countACK is used when broker ACK events can be simply forwarded to the
// producers ACKCount callback.
// The countACK is only applicable if no processors are configured.
// ACKs for closed clients will be ignored.
type countACK struct {
	active atomic.Bool
	fn     func(int)
}

// gapCountACK returns event ACKs to the producer, taking account for dropped events.
// Events being dropped by processors will always be ACKed with the last batch ACKed
// by the broker. This way clients waiting for ACKs can expect all processed
// events being alwyas ACKed.
type gapCountACK struct {
	fn func(int)

	active atomic.Bool
	done   chan struct{}
	drop   chan struct{}
	acks   chan int

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

// eventACK reports all ACKed events
type eventACK struct {
	mutex  sync.Mutex
	active bool

	// TODO: replace with more efficient dynamic sized ring-buffer?
	events []beat.Event

	acker acker

	fn func([]beat.Event)
}

// boundGapCountACK guards a gapCountACK instance by bounding the maximum number of
// active events.
// As beats might accumulate state while waiting for ACK, the boundGapCountACK blocks
// if too many events have been filtered out by processors.
type boundGapCountACK struct {
	active bool
	fn     func(int)

	acker *gapCountACK

	// simulate cancellable counting semaphore using counter + mutex + cond
	mutex      sync.Mutex
	cond       sync.Cond
	count, max int
}

// ACKer waiting for events being ACKed on shutdown
type waitACK struct {
	acker acker

	signal    chan struct{}
	waitClose time.Duration

	active atomic.Bool

	// number of active events
	events atomic.Uint64
}

// pipelineACK forwards event ACKs to the pipeline for
// global event accounting.
// Only overwrites ackEvents. The events counter must be incremented by the client,
// in case the event has been dropped by broker on TryPublish.
type pipelineACK struct {
	acker
	pipeline *Pipeline
}

func (*emptyACK) close()                        {}
func (*emptyACK) addEvent(_ beat.Event, _ bool) {}
func (*emptyACK) ackEvents(_ int)               {}

func makeCountACK(canDrop bool, max int, waitClose time.Duration, fn func(int)) acker {
	var acker acker
	if canDrop {
		acker = newBoundGapCountACK(max, fn)
	} else {
		acker = newCountACK(fn)
	}

	if waitClose <= 0 {
		return acker
	}

	wait := &waitACK{
		acker:     acker,
		signal:    make(chan struct{}, 1),
		waitClose: waitClose,
	}
	wait.active.Store(true)

	return wait
}

func newCountACK(fn func(int)) *countACK {
	a := &countACK{fn: fn}
	a.active.Store(true)
	return a
}

func (a *countACK) close()                        { a.active.Store(false) }
func (a *countACK) addEvent(_ beat.Event, _ bool) {}
func (a *countACK) ackEvents(n int) {
	if a.active.Load() {
		a.fn(n)
	}
}

func newGapCountACK(fn func(int)) *gapCountACK {
	a := &gapCountACK{
		fn:   fn,
		done: make(chan struct{}),
		drop: make(chan struct{}),
		acks: make(chan int, 1),
	}
	a.active.Store(true)

	init := &gapInfo{}
	a.lst.head = init
	a.lst.tail = init

	go a.ackLoop()
	return a
}

func (a *gapCountACK) ackLoop() {
	for {
		select {
		case <-a.done:
			return
		case n := <-a.acks:
			a.handleACK(n)
		case <-a.drop:
			// TODO: accumulate mulitple drop events + flush count with timer
			a.fn(1)
		}
	}
}

func (a *gapCountACK) handleACK(n int) {
	// collect items and compute total count from gapList
	total := n
	for n > 0 {
		a.lst.Lock()
		current := a.lst.head
		if n >= current.send {
			if current.next != nil {
				// advance list all event in current entry have been send and list as
				// more then 1 gapInfo entry. If only 1 entry is present, list item will be
				// reset and reused
				a.lst.head = current.next
			}
		}

		// hand over lock list-entry, so ACK handler and producer can operate
		// on potentially different list ends
		current.Lock()
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

	a.fn(total)
}

func (a *gapCountACK) close() {
	if a.active.Load() {
		close(a.done)
		a.active.Store(false)
	}
}

func (a *gapCountACK) addEvent(_ beat.Event, published bool) {
	if !a.active.Load() {
		return
	}

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
	if !a.active.Load() {
		return
	}

	select {
	case <-a.done:
	case a.acks <- n:
	}
}

func newBoundGapCountACK(max int, fn func(int)) *boundGapCountACK {
	a := &boundGapCountACK{active: true, max: max, fn: fn}
	a.cond.L = &a.mutex
	a.acker = newGapCountACK(a.onACK)
	return a
}

func (a *boundGapCountACK) close() {
	a.mutex.Lock()
	a.active = false
	a.cond.Broadcast()
	a.mutex.Unlock()
}

func (a *boundGapCountACK) addEvent(event beat.Event, published bool) {
	a.mutex.Lock()
	// block until some more 'space' has become available
	for a.active && a.count == a.max {
		a.cond.Wait()
	}

	a.count++
	active := a.active
	a.mutex.Unlock()

	if active {
		a.acker.addEvent(event, published)
	}
}

func (a *boundGapCountACK) ackEvents(n int) { a.acker.ackEvents(n) }
func (a *boundGapCountACK) onACK(n int) {
	a.mutex.Lock()

	old := a.count
	a.count -= n
	if old == a.max {
		a.cond.Broadcast()
	}

	a.mutex.Unlock()

	a.fn(n)
}

func newEventACK(canDrop bool, max int, waitClose time.Duration, fn func([]beat.Event)) *eventACK {
	a := &eventACK{fn: fn}
	a.active = true
	a.acker = makeCountACK(canDrop, max, waitClose, a.onACK)
	return a
}

func (a *eventACK) close() {
	a.mutex.Lock()
	a.mutex.Unlock()

	a.active = false
	a.events = nil

	a.acker.close()
}

func (a *eventACK) addEvent(event beat.Event, published bool) {
	a.mutex.Lock()
	a.events = append(a.events, event)
	a.mutex.Unlock()

	a.acker.addEvent(event, published)
}

func (a *eventACK) ackEvents(n int) {
	a.acker.ackEvents(n)
}

func (a *eventACK) onACK(n int) {
	a.mutex.Lock()
	if !a.active {
		a.mutex.Unlock()
		return
	}

	events := a.events[:n]
	a.events = a.events[n:]
	a.mutex.Unlock()

	if len(events) > 0 { // should always be true, just some safety-net
		a.fn(events)
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

func (a *pipelineACK) ackEvents(n int) {
	a.acker.ackEvents(n)
	a.pipeline.activeEventsDone(n)
}

func lastEventACK(fn func(beat.Event)) func([]beat.Event) {
	return func(events []beat.Event) {
		fn(events[len(events)-1])
	}
}
