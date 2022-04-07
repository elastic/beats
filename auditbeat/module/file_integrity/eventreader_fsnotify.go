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

//go:build linux || freebsd || openbsd || netbsd || windows
// +build linux freebsd openbsd netbsd windows

package file_integrity

import (
	"fmt"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/elastic/beats/v8/auditbeat/module/file_integrity/monitor"
	"github.com/elastic/beats/v8/libbeat/logp"
)

type reader struct {
	watcher monitor.Watcher
	config  Config
	eventC  chan Event
	log     *logp.Logger
}

// NewEventReader creates a new EventProducer backed by fsnotify.
func NewEventReader(c Config) (EventProducer, error) {
	return &reader{
		config: c,
		log:    logp.NewLogger(moduleName),
	}, nil
}

func (r *reader) Start(done <-chan struct{}) (<-chan Event, error) {
	watcher, err := monitor.New(r.config.Recursive, r.config.IsExcludedPath)
	if err != nil {
		return nil, err
	}

	r.watcher = watcher
	if err := r.watcher.Start(); err != nil {
		// Ensure that watcher is closed so that we don't leak watchers
		r.watcher.Close()
		return nil, fmt.Errorf("unable to start watcher: %w", err)
	}

	queueDone := make(chan struct{})
	queueC := make(chan []*Event)

	// Launch a separate goroutine to fetch all events that happen while
	// watches are being installed.
	go func() {
		defer close(queueC)
		queueC <- r.enqueueEvents(queueDone)
	}()

	// Windows implementation of fsnotify needs to have the watched paths
	// installed after the event consumer is started, to avoid a potential
	// deadlock. Do it on all platforms for simplicity.
	for _, p := range r.config.Paths {
		if err := r.watcher.Add(p); err != nil {
			if err == syscall.EMFILE {
				r.log.Warnw("Failed to add watch (check the max number of "+
					"open files allowed with 'ulimit -a')",
					"file_path", p, "error", err)
			} else {
				r.log.Warnw("Failed to add watch", "file_path", p, "error", err)
			}
		}
	}

	close(queueDone)
	events := <-queueC

	// Populate callee's event channel with the previously received events
	r.eventC = make(chan Event, 1+len(events))
	for _, ev := range events {
		r.eventC <- *ev
	}

	go r.consumeEvents(done)

	r.log.Infow("Started fsnotify watcher",
		"file_path", r.config.Paths,
		"recursive", r.config.Recursive)
	return r.eventC, nil
}

func (r *reader) enqueueEvents(done <-chan struct{}) (events []*Event) {
	for {
		ev := r.nextEvent(done)
		if ev == nil {
			return
		}
		events = append(events, ev)
	}
}

func (r *reader) consumeEvents(done <-chan struct{}) {
	defer close(r.eventC)
	defer r.watcher.Close()

	for {
		ev := r.nextEvent(done)
		if ev == nil {
			r.log.Debug("fsnotify reader terminated")
			return
		}
		r.eventC <- *ev
	}
}

func (r *reader) nextEvent(done <-chan struct{}) *Event {
	for {
		select {
		case <-done:
			return nil

		case event := <-r.watcher.EventChannel():
			if event.Name == "" || r.config.IsExcludedPath(event.Name) ||
				!r.config.IsIncludedPath(event.Name) {
				continue
			}
			r.log.Debugw("Received fsnotify event",
				"file_path", event.Name,
				"event_flags", event.Op)

			abs, err := filepath.Abs(event.Name)
			if err != nil {
				r.log.Errorw("Failed to obtain absolute path",
					"file_path", event.Name,
					"error", err,
				)
				event.Name = filepath.Clean(event.Name)
			} else {
				event.Name = abs
			}

			start := time.Now()
			e := NewEvent(event.Name, opToAction(event.Op), SourceFSNotify,
				r.config.MaxFileSizeBytes, r.config.HashTypes)
			e.rtt = time.Since(start)

			return &e

		case err := <-r.watcher.ErrorChannel():
			// a bug in fsnotify can cause spurious nil errors to be sent
			// on the error channel.
			if err != nil {
				r.log.Warnw("fsnotify watcher error", "error", err)
			}
		}
	}
}

func opToAction(op fsnotify.Op) Action {
	switch op {
	case fsnotify.Create:
		return Created
	case fsnotify.Write:
		return Updated
	case fsnotify.Remove:
		return Deleted
	case fsnotify.Rename:
		return Moved
	case fsnotify.Chmod:
		return AttributesModified
	default:
		return 0
	}
}
