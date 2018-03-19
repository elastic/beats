package channel

import (
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/common/atomic"
)

type subOutlet struct {
	isOpen atomic.Bool
	done   chan struct{}
	ch     chan *util.Data
	res    chan bool
}

// SubOutlet create a sub-outlet, which can be closed individually, without closing the
// underlying outlet.
func SubOutlet(out Outleter) Outleter {
	s := &subOutlet{
		isOpen: atomic.MakeBool(true),
		done:   make(chan struct{}),
		ch:     make(chan *util.Data),
		res:    make(chan bool, 1),
	}

	go func() {
		for event := range s.ch {
			s.res <- out.OnEvent(event)
		}
	}()

	return s
}

func (o *subOutlet) Close() error {
	isOpen := o.isOpen.Swap(false)
	if isOpen {
		close(o.done)
	}
	return nil
}

func (o *subOutlet) OnEvent(d *util.Data) bool {
	if !o.isOpen.Load() {
		return false
	}

	select {
	case <-o.done:
		close(o.ch)
		return false

	case o.ch <- d:
		select {
		case <-o.done:

			// Note: log harvester specific (leaky abstractions).
			//  The close at this point in time indicates an event
			//  already send to the publisher worker, forwarding events
			//  to the publisher pipeline. The harvester insists on updating the state
			//  (by pushing another state update to the publisher pipeline) on shutdown
			//  and requires most recent state update in the harvester (who can only
			//  update state on 'true' response).
			//  The state update will appear after the current event in the publisher pipeline.
			//  That is, by returning true here, the final state update will
			//  be presented to the reigstrar, after the last event being processed.
			//  Once all messages are in the publisher pipeline, in correct order,
			//  it depends on registrar/publisher pipeline if state is finally updated
			//  in the registrar.

			close(o.ch)
			return true

		case ret := <-o.res:
			return ret
		}
	}
}

// CloseOnSignal closes the outlet, once the signal triggers.
func CloseOnSignal(outlet Outleter, sig <-chan struct{}) Outleter {
	if sig != nil {
		go func() {
			<-sig
			outlet.Close()
		}()
	}
	return outlet
}
