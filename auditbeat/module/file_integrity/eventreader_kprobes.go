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

package file_integrity

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/elastic/beats/v7/auditbeat/module/file_integrity/kprobes"

	"github.com/elastic/elastic-agent-libs/logp"

	"golang.org/x/sys/unix"
)

type kProbesReader struct {
	watcher *kprobes.Monitor
	config  Config
	eventC  chan Event
	log     *logp.Logger

	parsers []FileParser
}

func (r kProbesReader) Start(done <-chan struct{}) (<-chan Event, error) {
	watcher, err := kprobes.New(r.config.Recursive)
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

	// kProbes watcher needs to have the watched paths
	// installed after the event consumer is started, to avoid a potential
	// deadlock. Do it on all platforms for simplicity.
	for _, p := range r.config.Paths {
		if err := r.watcher.Add(p); err != nil {
			if errors.Is(err, unix.EMFILE) {
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

	r.log.Infow("Started kprobes watcher",
		"file_path", r.config.Paths,
		"recursive", r.config.Recursive)
	return r.eventC, nil
}

func (r kProbesReader) enqueueEvents(done <-chan struct{}) []*Event {
	var events []*Event //nolint:prealloc //can't be preallocated as the number of events is unknown
	for {
		ev := r.nextEvent(done)
		if ev == nil {
			break
		}
		events = append(events, ev)
	}

	return events
}

func (r kProbesReader) consumeEvents(done <-chan struct{}) {
	defer close(r.eventC)
	defer r.watcher.Close()

	for {
		ev := r.nextEvent(done)
		if ev == nil {
			r.log.Debug("kprobes reader terminated")
			return
		}
		r.eventC <- *ev
	}
}

func (r kProbesReader) nextEvent(done <-chan struct{}) *Event {
	for {
		select {
		case <-done:
			return nil

		case event := <-r.watcher.EventChannel():
			if event.Path == "" || r.config.IsExcludedPath(event.Path) ||
				!r.config.IsIncludedPath(event.Path) {
				continue
			}
			r.log.Debugw("Received kprobes event",
				"file_path", event.Path,
				"event_flags", event.Op)

			abs, err := filepath.Abs(event.Path)
			if err != nil {
				r.log.Errorw("Failed to obtain absolute path",
					"file_path", event.Path,
					"error", err,
				)
				event.Path = filepath.Clean(event.Path)
			} else {
				event.Path = abs
			}

			start := time.Now()
			e := NewEvent(event.Path, kProbeTypeToAction(event.Op), SourceKProbes,
				r.config.MaxFileSizeBytes, r.config.HashTypes, r.parsers)
			e.rtt = time.Since(start)

			return &e

		case err := <-r.watcher.ErrorChannel():
			if err != nil {
				r.log.Errorw("kprobes watcher error", "error", err)
			}
		}
	}
}

func kProbeTypeToAction(op uint32) Action {
	switch op {
	case unix.IN_CREATE, unix.IN_MOVED_TO:
		return Created
	case unix.IN_MODIFY:
		return Updated
	case unix.IN_DELETE:
		return Deleted
	case unix.IN_MOVED_FROM:
		return Moved
	case unix.IN_ATTRIB:
		return AttributesModified
	default:
		return 0
	}
}
