package channel

import (
	"sync"
	"sync/atomic"

	"github.com/elastic/beats/filebeat/input"
)

type Outlet struct {
	wg      *sync.WaitGroup
	done    <-chan struct{}
	signal  <-chan struct{}
	channel chan *input.Event
	isOpen  int32 // atomic indicator
}

func NewOutlet(
	done <-chan struct{},
	c chan *input.Event,
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
func (o *Outlet) SetSignal(signal <-chan struct{}) {
	o.signal = signal
}

func (o *Outlet) OnEvent(event *input.Event) bool {
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
	case o.channel <- event:
		return true
	}
}

// OnEventSignal can be stopped by the signal that is set with SetSignal
// This does not close the outlet. Only OnEvent does close the outlet.
func (o *Outlet) OnEventSignal(event *input.Event) bool {
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
	case o.channel <- event:
		return true
	}
}
