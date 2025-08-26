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

package es

import (
	"sync"

	conf "github.com/elastic/elastic-agent-libs/config"
)

type OnConfigUpdateFunc func(c *conf.C)
type UnsubscribeFunc func()

type Notifier struct {
	mx sync.Mutex

	lastConfig *conf.C
	listeners  map[int]OnConfigUpdateFunc
	id         int
}

func NewNotifier() *Notifier {
	return &Notifier{
		listeners: make(map[int]OnConfigUpdateFunc),
		id:        0,
	}
}

// Subscribe adds a listener to the notifier. The listener will be called when Notify is called.
// Each OnConfigUpdateFunc is called asynchronously in a separate goroutine in each Notify call.
//
// Returns an UnsubscribeFunc that can be used to remove the listener.
//
// Note: Subscribe will call the listener with the last config that was passed to Notify.
func (n *Notifier) Subscribe(fn OnConfigUpdateFunc) UnsubscribeFunc {
	n.mx.Lock()
	defer n.mx.Unlock()

	id := n.id
	n.id++
	n.listeners[id] = fn

	if n.lastConfig != nil {
		go fn(n.lastConfig)
	}

	return func() {
		n.mx.Lock()
		defer n.mx.Unlock()
		delete(n.listeners, id)
	}
}

func (n *Notifier) Notify(c *conf.C) {
	n.mx.Lock()
	defer n.mx.Unlock()
	n.lastConfig = c

	for _, listener := range n.listeners {
		go listener(c)
	}
}
