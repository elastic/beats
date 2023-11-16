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

	go r.consumeEvents()

	r.log.Infow("started ebpf watcher", "file_path", r.config.Paths, "recursive", r.config.Recursive)
	return r.eventC, nil
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
