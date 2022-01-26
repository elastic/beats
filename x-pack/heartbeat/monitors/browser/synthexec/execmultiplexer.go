// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package synthexec

import (
	"github.com/elastic/beats/v7/libbeat/common/atomic"
)

type ExecMultiplexer struct {
	currentJourney *atomic.Bool
	eventCounter   *atomic.Int
	synthEvents    chan *SynthEvent
	done           chan struct{}
}

func (e ExecMultiplexer) Close() {
	close(e.synthEvents)
}

func (e ExecMultiplexer) writeSynthEvent(se *SynthEvent) {
	if se == nil { // we skip writing nil events, since a nil means we're done
		return
	}

	if se.Type == "journey/start" {
		e.currentJourney.Store(true)
		e.eventCounter.Store(-1)
	}
	se.index = e.eventCounter.Inc()

	if se.Type == "journey/end" || se.Type == "cmd/status" {
		e.currentJourney.Store(false)
	}
	e.synthEvents <- se
}

// SynthEvents returns a read only channel for synth events
func (e ExecMultiplexer) SynthEvents(inline bool) <-chan *SynthEvent {
	if inline || e.currentJourney.Load() {
		return e.synthEvents
	}
	return make(chan *SynthEvent)
}

// Done returns a channel that is closed when all output has been received
func (e ExecMultiplexer) Done() <-chan struct{} {
	return e.done
}

// Wait blocks until the multiplexer is done and has returned all data
func (e ExecMultiplexer) Wait() {
	<-e.done
}

func NewExecMultiplexer() *ExecMultiplexer {
	return &ExecMultiplexer{
		currentJourney: atomic.NewBool(false),
		eventCounter:   atomic.NewInt(-1), // Start from -1 so first call to Inc returns 0
		synthEvents:    make(chan *SynthEvent),
		done:           make(chan struct{}),
	}
}
