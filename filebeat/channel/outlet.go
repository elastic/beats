package channel

import (
	"sync"
	"sync/atomic"

	"github.com/elastic/beats/filebeat/util"
)

// Outlet struct is used to be passed to an object which needs an outlet
//
// The difference between signal and done channel is as following:
// - signal channel can be added through SetSignal and is used to
//   interrupt events sent through OnEventSignal-
// - done channel is used to close and stop the outlet
//
// If SetSignal is used, it must be ensure that there is only one event producer.
type Outlet struct {
	wg      *sync.WaitGroup // Use for counting active events
	done    <-chan struct{}
	signal  <-chan struct{}
	channel chan *util.Data
	isOpen  int32 // atomic indicator
}

func NewOutlet(
	done <-chan struct{},
	c chan *util.Data,
	wg *sync.WaitGroup,
) *Outlet {
	return &Outlet{
		done:    done,
		channel: c,
		wg:      wg,
		isOpen:  1,
	}
}

// SetSignal sets the signal channel for OnEventSignal
// If SetSignal is used, it must be ensure that only one producer exists.
func (o *Outlet) SetSignal(signal <-chan struct{}) {
	o.signal = signal
}

func (o *Outlet) OnEvent(data *util.Data) bool {
	open := atomic.LoadInt32(&o.isOpen) == 1
	if !open {
		return false
	}

	if o.wg != nil {
		o.wg.Add(1)
	}

	select {
	case <-o.done:
		if o.wg != nil {
			o.wg.Done()
		}
		atomic.StoreInt32(&o.isOpen, 0)
		return false
	case o.channel <- data:
		return true
	}
}

// OnEventSignal can be stopped by the signal that is set with SetSignal
// This does not close the outlet. Only OnEvent does close the outlet.
// If OnEventSignal is used, it must be ensured that only one producer is used.
func (o *Outlet) OnEventSignal(data *util.Data) bool {
	open := atomic.LoadInt32(&o.isOpen) == 1
	if !open {
		return false
	}

	if o.wg != nil {
		o.wg.Add(1)
	}

	select {
	case <-o.signal:
		if o.wg != nil {
			o.wg.Done()
		}
		o.signal = nil
		return false
	case o.channel <- data:
		return true
	}
}

func (o *Outlet) Copy() Outleter {
	return NewOutlet(o.done, o.channel, o.wg)
}
