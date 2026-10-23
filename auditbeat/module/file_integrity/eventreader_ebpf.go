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

//go:build linux && (amd64 || arm64) && cgo

package file_integrity

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	quark "github.com/elastic/go-quark"

	"github.com/elastic/elastic-agent-libs/logp"
)

// ebpfReader receives file events from the eBPF backend in quark.
type ebpfReader struct {
	config  Config
	log     *logp.Logger
	eventC  chan Event
	parsers []FileParser
	paths   map[string]struct{}
	queue   *quark.Queue
}

func (r *ebpfReader) Start(done <-chan struct{}) (<-chan Event, error) {
	attr := quark.DefaultQueueAttr()
	// Only the eBPF backend in quark produces file events, so there is no
	// fallback to quark's kprobe backend. Kernels without eBPF are served
	// by this module's own kprobes backend instead.
	attr.Flags = quark.QQ_EBPF | quark.QQ_FILE

	queue, err := quark.OpenQueue(attr)
	if err != nil {
		return nil, fmt.Errorf("open quark queue: %w", err)
	}
	r.queue = queue

	go r.consumeEvents(done)

	r.log.Infow("started ebpf watcher", "file_path", r.config.Paths, "recursive", r.config.Recursive)
	return r.eventC, nil
}

// consumeEvents owns the quark queue. The queue is not safe for concurrent
// use, so every call on it, including Close, happens on this goroutine.
func (r *ebpfReader) consumeEvents(done <-chan struct{}) {
	defer close(r.eventC)
	defer r.queue.Close()

	var lost uint64
	for {
		select {
		case <-done:
			r.log.Debug("ebpf watcher terminated")
			return
		default:
		}

		qe, ok := r.queue.GetEvent()
		if !ok {
			// Block returns after at most 100ms so done is re-checked promptly.
			if err := r.queue.Block(); err != nil {
				r.log.Errorf("ebpf watcher error: %v", err)
				return
			}
			if stats := r.queue.Stats(); stats.Lost != lost {
				r.log.Warnf("ebpf watcher lost %d events", stats.Lost-lost)
				lost = stats.Lost
			}
			continue
		}

		if qe.Events&quark.QUARK_EV_FILE == 0 || qe.File == nil {
			r.log.Debugf("received unwanted quark event: %#x", qe.Events)
			continue
		}

		start := time.Now()
		e, ok := NewEventFromQuarkEvent(
			qe,
			r.config.MaxFileSizeBytes,
			r.config.HashTypes,
			r.parsers,
			r.excludedPath,
		)
		if !ok {
			continue
		}
		e.rtt = time.Since(start)

		r.log.Debugw("received ebpf event", "file_path", e.Path)
		select {
		case r.eventC <- e:
		case <-done:
			r.log.Debug("ebpf watcher terminated")
			return
		}
	}
}

func (r *ebpfReader) excludedPath(path string) bool {
	dir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		r.log.Errorf("ebpf watcher error: resolve abs path %q: %v", path, err)
		return true
	}

	if r.config.IsExcludedPath(dir) {
		return true
	}

	if !r.config.Recursive {
		if _, ok := r.paths[dir]; ok {
			return false
		}
	} else {
		for p := range r.paths {
			if strings.HasPrefix(dir, p) {
				return false
			}
		}
	}

	return true
}
