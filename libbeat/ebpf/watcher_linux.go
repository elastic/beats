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

//go:build linux

package ebpf

import (
	"context"
	"fmt"
	"sync"

	"github.com/elastic/ebpfevents"
)

var (
	gWatcherOnce sync.Once
	gWatcher     Watcher
)

type client struct {
	name    string
	mask    EventMask
	records chan ebpfevents.Record
}

// EventMask is a mask of ebpfevents.EventType which is used to control which event types clients will receive.
type EventMask uint64

// Watcher observes kernel events, using ebpf probes from the ebpfevents library, and sends the
// events to subscribing clients.
//
// A single global watcher can exist, and can deliver events to multiple clients. Clients subscribe
// to the watcher, and all ebpf events that match their mask will be sent to their channel.
type Watcher struct {
	sync.Mutex
	cancel  context.CancelFunc
	loader  *ebpfevents.Loader
	clients map[string]client
	status  status
	err     error
}

type status int

const (
	stopped status = iota
	started
)

// GetWatcher creates the watcher, if required, and returns a reference to the global Watcher.
func GetWatcher() (*Watcher, error) {
	gWatcher.Lock()
	defer gWatcher.Unlock()

	// Try to load the probe once on startup so consumers can error out.
	gWatcherOnce.Do(func() {
		if gWatcher.status == stopped {
			l, err := ebpfevents.NewLoader()
			if err != nil {
				gWatcher.err = fmt.Errorf("init ebpf loader: %w", err)
				return
			}
			_ = l.Close()
		}
	})

	return &gWatcher, gWatcher.err
}

// Subscribe to receive events from the watcher.
func (w *Watcher) Subscribe(clientName string, events EventMask) <-chan ebpfevents.Record {
	w.Lock()
	defer w.Unlock()

	if w.status == stopped {
		w.startLocked()
	}

	w.clients[clientName] = client{
		name:    clientName,
		mask:    events,
		records: make(chan ebpfevents.Record, w.loader.BufferLen()),
	}

	return w.clients[clientName].records
}

// Unsubscribe the client with the given name.
func (w *Watcher) Unsubscribe(clientName string) {
	w.Lock()
	defer w.Unlock()

	delete(w.clients, clientName)

	if w.nclients() == 0 {
		w.stopLocked()
	}
}

func (w *Watcher) startLocked() {
	if w.status == started {
		return
	}

	loader, err := ebpfevents.NewLoader()
	if err != nil {
		w.err = fmt.Errorf("start ebpf loader: %w", err)
		return
	}

	w.loader = loader
	w.clients = make(map[string]client)

	records := make(chan ebpfevents.Record, loader.BufferLen())
	var ctx context.Context
	ctx, w.cancel = context.WithCancel(context.Background())

	go w.loader.EventLoop(ctx, records)
	go func(ctx context.Context) {
		for {
			select {
			case record := <-records:
				if record.Error != nil {
					for _, client := range w.clients {
						client.records <- record
					}
					continue
				}
				for _, client := range w.clients {
					if client.mask&EventMask(record.Event.Type) != 0 {
						client.records <- record
					}
				}
				continue
			case <-ctx.Done():
				return
			}
		}
	}(ctx)

	w.status = started
}

func (w *Watcher) stopLocked() {
	if w.status == stopped {
		return
	}
	w.close()
	w.status = stopped
}

func (w *Watcher) nclients() int {
	return len(w.clients)
}

func (w *Watcher) close() {
	if w.cancel != nil {
		w.cancel()
	}

	if w.loader != nil {
		_ = w.loader.Close()
	}

	for _, cl := range w.clients {
		close(cl.records)
	}
}
