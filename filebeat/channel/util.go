// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package channel

import (
	"sync"

	"github.com/elastic/beats/v8/libbeat/beat"
)

type subOutlet struct {
	done      chan struct{}
	ch        chan beat.Event
	res       chan bool
	mutex     sync.Mutex
	closeOnce sync.Once
}

// SubOutlet create a sub-outlet, which can be closed individually, without closing the
// underlying outlet.
func SubOutlet(out Outleter) Outleter {
	s := &subOutlet{
		done: make(chan struct{}),
		ch:   make(chan beat.Event),
		res:  make(chan bool, 1),
	}

	go func() {
		for event := range s.ch {
			s.res <- out.OnEvent(event)
		}
	}()

	return s
}

func (o *subOutlet) Close() error {
	o.closeOnce.Do(func() {
		// Signal OnEvent() to terminate
		close(o.done)
		// This mutex prevents the event channel to be closed if OnEvent is
		// still running.
		o.mutex.Lock()
		defer o.mutex.Unlock()
		close(o.ch)
	})
	return nil
}

func (o *subOutlet) Done() <-chan struct{} {
	return o.done
}

func (o *subOutlet) OnEvent(event beat.Event) bool {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	select {
	case <-o.done:
		return false
	default:
	}

	select {
	case <-o.done:
		return false

	case o.ch <- event:
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
			//  be presented to the registrar, after the last event being processed.
			//  Once all messages are in the publisher pipeline, in correct order,
			//  it depends on registrar/publisher pipeline if state is finally updated
			//  in the registrar.
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
			select {
			case <-outlet.Done():
				return
			case <-sig:
				outlet.Close()
			}
		}()
	}
	return outlet
}
