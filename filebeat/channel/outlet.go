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
	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common/atomic"
)

type outlet struct {
	client beat.Client
	isOpen atomic.Bool
	done   chan struct{}
}

func newOutlet(client beat.Client) *outlet {
	o := &outlet{
		client: client,
		isOpen: atomic.MakeBool(true),
		done:   make(chan struct{}),
	}
	return o
}

func (o *outlet) Close() error {
	isOpen := o.isOpen.Swap(false)
	if isOpen {
		close(o.done)
		return o.client.Close()
	}
	return nil
}

func (o *outlet) Done() <-chan struct{} {
	return o.done
}

func (o *outlet) OnEvent(event beat.Event) bool {
	if !o.isOpen.Load() {
		return false
	}

	o.client.Publish(event)

	// Note: race condition on shutdown:
	//  The underlying beat.Client is asynchronous. Without proper ACK
	//  handler we can not tell if the event made it 'through' or the client
	//  close has been completed before sending. In either case,
	//  we report 'false' here, indicating the event eventually being dropped.
	//  Returning false here, prevents the harvester from updating the state
	//  to the most recently published events. Therefore, on shutdown the harvester
	//  might report an old/outdated state update to the registry, overwriting the
	//  most recently
	//  published offset in the registry on shutdown.
	return o.isOpen.Load()
}
