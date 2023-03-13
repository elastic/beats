// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux
// +build linux

package synthexec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/elastic/elastic-agent-libs/logp"
)

var traceableTypes = map[string]bool{
	JourneyStart: true,
	JourneyEnd:   true,
	CmdStatus:    true,
}

type SynthEventTracer interface {
	Write(event *SynthEvent)
	Done()
}

type synthEventTracer struct {
	ctx    context.Context
	cancel context.CancelFunc
	events chan *SynthEvent
	writer *os.File
}

func (s *synthEventTracer) Write(event *SynthEvent) {
	s.events <- event
}

func (s *synthEventTracer) Done() {
	s.cancel()
}

func (s *synthEventTracer) writeF() {
	defer s.writer.Close()
	for {
		select {
		case <-s.ctx.Done():
			return
		case event := <-s.events:
			if _, ok := traceableTypes[event.Type]; ok {
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
}

func NewSynthEventTracer(ctx context.Context, path string) SynthEventTracer {
	w, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0770)
	if err != nil {
		logp.L().Error("error opening trace file: %w", err)
		return NewNoopTracer()
	}

	ctx, cancel := context.WithCancel(ctx)
	s := &synthEventTracer{
		ctx:    ctx,
		cancel: cancel,
		events: make(chan *SynthEvent),
		writer: w,
	}

	go s.writeF()
	return s
}

type NoopTracer struct{}

func (NoopTracer) Write(event *SynthEvent) {}
func (NoopTracer) Done()                   {}

func NewNoopTracer() SynthEventTracer {
	return NoopTracer{}
}
