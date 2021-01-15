// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package synthexec

import (
	"encoding/json"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/logp"
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
	}
	hasCurrentJourney := e.currentJourney.Load()
	if se.Type == "journey/end" {
		e.currentJourney.Store(false)
	}

	out, err := json.Marshal(se)

	se.index = e.eventCounter.Inc()
	if hasCurrentJourney {
		e.synthEvents <- se
	} else {
		logp.Warn("received output from synthetics outside of journey scope: %s %s", out, err)
	}
}

// SynthEvents returns a read only channel for synth events
func (e ExecMultiplexer) SynthEvents() <-chan *SynthEvent {
	return e.synthEvents
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
