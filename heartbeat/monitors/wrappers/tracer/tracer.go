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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
)

type EventTracer interface {
	Write(event *beat.Event)
	Done()
}

type eventTracer struct {
	ctx    context.Context
	cancel context.CancelFunc
	events chan *beat.Event
	writer *os.File
}

func (t *eventTracer) Write(event *beat.Event) {
	t.events <- event
}

func (t *eventTracer) Done() {
	t.cancel()
}

func (s *eventTracer) writeF() {
	defer s.writer.Close()
	for {
		select {
		case <-s.ctx.Done():
			return
		case event := <-s.events:
			j, err := json.Marshal(event)
			if err != nil {
				logp.L().Error("Error marshalling event: %w", err)
			}

			_, err = fmt.Fprintf(s.writer, "%s\n", j)
			if err != nil {
				logp.L().Error("Error writing to trace file: %w", err)
			}

			s.writer.Sync()
		}
	}
}

func NewEventTracer(path string) EventTracer {
	w, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0770)
	if err != nil {
		logp.L().Error("error opening trace file: %w", err)
		return NewNoopTracer()
	}

	ctx, cancel := context.WithCancel(context.Background())
	s := &eventTracer{
		ctx:    ctx,
		cancel: cancel,
		events: make(chan *beat.Event),
		writer: w,
	}

	go s.writeF()
	return s
}

// Dummy noop tracer
type NoopTracer struct{}

func (*NoopTracer) Write(event *beat.Event) {}
func (*NoopTracer) Done()                   {}

func NewNoopTracer() EventTracer {
	return &NoopTracer{}
}
