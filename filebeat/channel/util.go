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
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/atomic"
)

type subOutlet struct {
	isOpen atomic.Bool
	done   chan struct{}
	ch     chan *util.Data
	res    chan bool
}

// ConnectTo creates a new Connector, combining a beat.Pipeline with an outlet Factory.
func ConnectTo(pipeline beat.Pipeline, factory Factory) Connector {
	return func(cfg *common.Config, m *common.MapStrPointer) (Outleter, error) {
		return factory(pipeline, cfg, m)
	}
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
