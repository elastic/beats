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

package tracer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
)

type EventTracer interface {
	Write(event *beat.Event)
	Done()
	GetFilter() []string
}

type eventTracer struct {
	ctx    context.Context
	cancel context.CancelFunc
	events chan *beat.Event
	writer *os.File
	filter []string
}

func (t *eventTracer) Write(event *beat.Event) {
	t.events <- event
}

func (t *eventTracer) Done() {
	t.cancel()
}

func (t *eventTracer) GetFilter() []string {
	return t.filter
}

func (t *eventTracer) writeF() {
	defer t.writer.Close()
	for {
		select {
		case <-t.ctx.Done():
			return
		case event := <-t.events:
			j, err := json.Marshal(event)
			if err != nil {
				logp.L().Error("Error marshalling event: %w", err)
			}

			_, err = fmt.Fprintf(t.writer, "%s\n", j)
			if err != nil {
				logp.L().Error("Error writing to trace file: %w", err)
			}

			err = t.writer.Sync()
			if err != nil {
				logp.L().Error("Error flushing trace file: %w", err)
			}
		}
	}
}

func NewEventTracer(path string, perms os.FileMode, filter []string) EventTracer {
	file := filepath.Base(path)
	dir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		logp.L().Error("error resolving trace path: %w", err)
		return NewNoopTracer()
	}

	err = os.MkdirAll(dir, perms)
	if err != nil {
		logp.L().Error("error creating trace path: %w", err)
		return NewNoopTracer()
	}

	w, err := os.OpenFile(filepath.Join(dir, file), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perms)
	if err != nil {
		logp.L().Error("error opening trace file: %w", err)
		return NewNoopTracer()
	}

	logp.L().Infof("trace file open at: %s, filtering for %v", filepath.Join(dir, file), filter)

	ctx, cancel := context.WithCancel(context.Background())
	s := &eventTracer{
		ctx:    ctx,
		cancel: cancel,
		events: make(chan *beat.Event),
		writer: w,
		filter: filter,
	}

	go s.writeF()
	return s
}

// Dummy noop tracer
type NoopTracer struct{}

func (*NoopTracer) Write(event *beat.Event) {}
func (*NoopTracer) Done()                   {}
func (*NoopTracer) GetFilter() []string     { return nil }

func NewNoopTracer() EventTracer {
	return &NoopTracer{}
}
