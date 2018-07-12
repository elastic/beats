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

package bus

import (
	"sync"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Event sent to the bus
type Event common.MapStr

// Bus provides a common channel to emit and listen for Events
type Bus interface {
	// Publish an event to the bus
	Publish(Event)

	// Subscribe to all events, filter them to the ones containing *all* the keys in filter
	Subscribe(filter ...string) Listener
}

// Listener retrieves Events from a Bus subscription until Stop is called
type Listener interface {
	// Events channel
	Events() <-chan Event

	// Stop listening and removes itself from the bus
	Stop()
}

type bus struct {
	sync.RWMutex
	name      string
	listeners []*listener
}

type listener struct {
	filter  []string
	channel chan Event
	bus     *bus
}

// New initializes a new bus with the given name and returns it
func New(name string) Bus {
	return &bus{
		name:      name,
		listeners: make([]*listener, 0),
	}
}

func (b *bus) Publish(e Event) {
	b.RLock()
	defer b.RUnlock()

	logp.Debug("bus", "%s: %+v", b.name, e)
	for _, listener := range b.listeners {
		if listener.interested(e) {
			listener.channel <- e
		}
	}
}

func (b *bus) Subscribe(filter ...string) Listener {
	listener := &listener{
		filter:  filter,
		bus:     b,
		channel: make(chan Event, 100),
	}

	b.Lock()
	defer b.Unlock()
	b.listeners = append(b.listeners, listener)

	return listener
}

func (l *listener) Events() <-chan Event {
	return l.channel
}

func (l *listener) Stop() {
	l.bus.Lock()
	defer l.bus.Unlock()

	for i, listener := range l.bus.listeners {
		if l == listener {
			l.bus.listeners = append(l.bus.listeners[:i], l.bus.listeners[i+1:]...)
		}
	}

	close(l.channel)
}

// Return true if listener is interested on the given event
func (l *listener) interested(e Event) bool {
	for _, key := range l.filter {
		if _, ok := e[key]; !ok {
			return false
		}
	}
	return true
}
