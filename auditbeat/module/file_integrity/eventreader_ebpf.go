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

//go:build linux && (amd64 || arm64)

package file_integrity

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/ebpf"
	"github.com/elastic/ebpfevents"
	"github.com/elastic/elastic-agent-libs/logp"
)

const clientName = "fim"

type ebpfReader struct {
	watcher *ebpf.Watcher
	done    <-chan struct{}
	config  Config
	log     *logp.Logger
	eventC  chan Event
	parsers []FileParser
	paths   map[string]struct{}

	_records <-chan ebpfevents.Record
}

func (r *ebpfReader) Start(done <-chan struct{}) (<-chan Event, error) {
	watcher, err := ebpf.GetWatcher()
	if err != nil {
		return nil, err
	}
	r.watcher = watcher
	r.done = done

	mask := ebpf.EventMask(ebpfevents.EventTypeFileCreate | ebpfevents.EventTypeFileRename | ebpfevents.EventTypeFileDelete | ebpfevents.EventTypeFileModify)
	r._records = r.watcher.Subscribe(clientName, mask)

	// The loader falls back when some of its programs fail to load on this kernel. It
	// keeps running, but events only the dropped programs emit are lost.
	degraded := r.watcher.Degraded()
	if degraded != nil {
		r.log.Warnw("ebpf loader fell back on this kernel", "error", degraded)
	}
	if missing := missingEvents(mask, r.watcher.DisabledEvents()); len(missing) > 0 {
		r.watcher.Unsubscribe(clientName)
		return nil, fmt.Errorf("ebpf events required by file_integrity are disabled on this kernel: %s: %w",
			strings.Join(missing, ", "), degraded)
	}

	go r.consumeEvents()

	r.log.Infow("started ebpf watcher", "file_path", r.config.Paths, "recursive", r.config.Recursive)
	return r.eventC, nil
}

// missingEvents returns the names of the event types in mask that are disabled.
func missingEvents(mask ebpf.EventMask, disabled []ebpfevents.EventType) []string {
	var missing []string
	for _, et := range disabled {
		if mask&ebpf.EventMask(et) != 0 {
			missing = append(missing, et.String())
		}
	}
	return missing
}

func (r *ebpfReader) consumeEvents() {
	defer close(r.eventC)
	defer r.watcher.Unsubscribe(clientName)

	for {
		select {
		case rec := <-r._records:
			if rec.Error != nil {
				r.log.Errorf("ebpf watcher error: %v", rec.Error)
				continue
			}

			switch rec.Event.Type {
			case ebpfevents.EventTypeFileCreate, ebpfevents.EventTypeFileRename, ebpfevents.EventTypeFileDelete, ebpfevents.EventTypeFileModify:
			default:
				r.log.Warnf("received unwanted ebpf event: %s", rec.Event.Type.String())
				continue
			}

			start := time.Now()
			e, ok := NewEventFromEbpfEvent(
				*rec.Event,
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
			r.eventC <- e
		case <-r.done:
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
