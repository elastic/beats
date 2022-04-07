// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package synthexec

import (
	"github.com/elastic/beats/v8/libbeat/common/atomic"
)

type ExecMultiplexer struct {
	eventCounter *atomic.Int
	synthEvents  chan *SynthEvent
	done         chan struct{}
}

func (e ExecMultiplexer) Close() {
	close(e.synthEvents)
}

func (e ExecMultiplexer) writeSynthEvent(se *SynthEvent) {
	if se == nil { // we skip writing nil events, since a nil means we're done
		return
	}

	if se.Type == "journey/start" {
		e.eventCounter.Store(-1)
	}
	se.index = e.eventCounter.Inc()

	e.synthEvents <- se
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
		eventCounter: atomic.NewInt(-1), // Start from -1 so first call to Inc returns 0
		synthEvents:  make(chan *SynthEvent),
		done:         make(chan struct{}),
	}
}
