package outputs

import (
	"github.com/elastic/libbeat/logp"
	"sync/atomic"
)

// Signaler signals the completion of potentially asynchronous output operation.
// Completed is called by the output plugin when all events have been sent. On
// failure or if only a subset of the data is published then Failed will be
// invoked.
type Signaler interface {
	Completed()

	Failed()
}

// ChanSignal will send outputer signals on configurable channel.
type ChanSignal struct {
	ch chan bool
}

// SyncSignal blocks waiting for a signal.
type SyncSignal struct {
	ch chan bool
}

// SplitSignal guards one output signaler from multiple calls
// by using a simple reference counting scheme. If one Signaler consumer
// reports a Failed event, the Failed event will be send to the guarded Signaler
// once the reference count becomes zero.
//
// Example use cases:
//   - Push signaler to multiple outputers
//   - split data to be send in smaller batches
type SplitSignal struct {
	count    int32
	failed   bool
	signaler Signaler
}

// CompositeSignal combines multiple signalers into one Signaler forwarding an event to
// to all signalers.
type CompositeSignal struct {
	signalers []Signaler
}

// NewChanSignal create a new ChanSignal forwarding signals to a channel.
func NewChanSignal(ch chan bool) *ChanSignal { return &ChanSignal{ch} }

// Completed sends true to the confiugred channel.
func (c *ChanSignal) Completed() { c.ch <- true }

// Failed sends false to the confiugred channel.
func (c *ChanSignal) Failed() { c.ch <- false }

// NewSyncSignal create a new SyncSignal signaler. Use Wait() method to wait for
// a signal from the publisher
func NewSyncSignal() *SyncSignal { return &SyncSignal{make(chan bool, 1)} }

// Wait blocks waiting for a signal from the outputer. Wait return true if
// Completed was signaled and false if a Failed signal was received
func (s *SyncSignal) Wait() bool { return <-s.ch }

// Completed sends true to the process waiting for a signal.
func (s *SyncSignal) Completed() { s.ch <- true }

// Failed sends false to the process waiting for a signal.
func (s *SyncSignal) Failed() { s.ch <- false }

// NewSplitSignaler creates a new SplitSignal if s is not nil.
// If s is nil, nil will be returned. The count is the number of events to be
// received before publishing the final event to the guarded Signaler.
func NewSplitSignaler(
	s Signaler,
	count int,
) Signaler {
	if s == nil {
		return nil
	}

	return &SplitSignal{
		count:    int32(count),
		signaler: s,
	}
}

// Completed signals a Completed event to s.
func (s *SplitSignal) Completed() {
	s.onEvent()
}

// Failed signals a Failed event to s.
func (s *SplitSignal) Failed() {
	s.failed = true
	s.onEvent()
}

func (s *SplitSignal) onEvent() {
	res := atomic.AddInt32(&s.count, -1)
	if res == 0 {
		if s.failed {
			s.signaler.Failed()
		} else {
			s.signaler.Completed()
		}
	}
}

// NewCompositeSignaler creates a new composite signaler.
func NewCompositeSignaler(signalers ...Signaler) Signaler {
	if len(signalers) == 0 {
		return nil
	}
	return &CompositeSignal{signalers}
}

// Completed sends the Completed signal to all signalers.
func (cs *CompositeSignal) Completed() {
	for _, s := range cs.signalers {
		if s != nil {
			s.Completed()
		}
	}
}

// Failed sends the Failed signal to all signalers.
func (cs *CompositeSignal) Failed() {
	for _, s := range cs.signalers {
		if s != nil {
			s.Failed()
		}
	}
}

// SignalCompleted sends the Completed event to s if s is not nil.
func SignalCompleted(s Signaler) {
	if s != nil {
		s.Completed()
	}
}

// SignalFailed sends the Failed event to s if s is not nil
func SignalFailed(s Signaler, err error) {

	if err != nil {
		logp.Err("Error sending/writing event: %s", err)
	}

	if s != nil {
		s.Failed()
	}
}

// Signal will send the Completed or Failed event to s depending
// on err being set if s is not nil.
func Signal(s Signaler, err error) {

	if err != nil {
		logp.Info("Failed to send event %s", err)
	}

	if s != nil {
		if err == nil {
			s.Completed()
		} else {
			s.Failed()
		}
	}
}

// SignalAll send the Completed or Failed event to all given signalers
// depending on err being set.
func SignalAll(signalers []Signaler, err error) {
	if signalers != nil {
		Signal(NewCompositeSignaler(signalers...), err)
	}
}
