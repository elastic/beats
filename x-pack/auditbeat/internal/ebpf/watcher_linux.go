// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

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
	gWatcherErr  error
	gWatcher     watcher
)

type client struct {
	name    string
	mask    EventMask
	records chan ebpfevents.Record
}

type watcher struct {
	sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
	loader  *ebpfevents.Loader
	clients map[string]client
	status  status
}

type status int

const (
	stopped status = iota
	started
)

func GetWatcher() (Watcher, error) {
	gWatcher.Lock()
	defer gWatcher.Unlock()

	// Try to load the probe once on startup so consumers can error out.
	gWatcherOnce.Do(func() {
		if gWatcher.status == stopped {
			l, err := ebpfevents.NewLoader()
			if err != nil {
				gWatcherErr = fmt.Errorf("init ebpf loader: %w", err)
				return
			}
			_ = l.Close()
		}
	})

	return &gWatcher, gWatcherErr
}

func (w *watcher) Subscribe(name string, events EventMask) <-chan ebpfevents.Record {
	w.Lock()
	defer w.Unlock()

	if w.status == stopped {
		startLocked()
	}

	w.clients[name] = client{
		name:    name,
		mask:    events,
		records: make(chan ebpfevents.Record),
	}

	return w.clients[name].records
}

func (w *watcher) Unsubscribe(name string) {
	w.Lock()
	defer w.Unlock()

	delete(w.clients, name)

	if w.nclients() == 0 {
		stopLocked()
	}
}

func startLocked() {
	loader, err := ebpfevents.NewLoader()
	if err != nil {
		gWatcherErr = fmt.Errorf("start ebpf loader: %w", err)
		return
	}

	gWatcher.loader = loader
	gWatcher.clients = make(map[string]client)

	records := make(chan ebpfevents.Record, loader.BufferLen())
	gWatcher.ctx, gWatcher.cancel = context.WithCancel(context.Background())

	go gWatcher.loader.EventLoop(gWatcher.ctx, records)
	go func() {
		for {
			select {
			case record := <-records:
				if record.Error != nil {
					for _, client := range gWatcher.clients {
						client.records <- record
					}
					continue
				}
				for _, client := range gWatcher.clients {
					if client.mask&EventMask(record.Event.Type) != 0 {
						client.records <- record
					}
				}
				continue
			case <-gWatcher.ctx.Done():
				return
			}
		}
	}()

	gWatcher.status = started
}

func stopLocked() {
	_ = gWatcher.close()
	gWatcher.status = stopped
}

func (w *watcher) nclients() int {
	return len(w.clients)
}

func (w *watcher) close() error {
	if w.cancel != nil {
		w.cancel()
	}

	if w.loader != nil {
		_ = w.loader.Close()
	}

	for _, cl := range w.clients {
		close(cl.records)
	}

	return nil
}
