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

package synthexec

import (
	"github.com/elastic/beats/v7/libbeat/common/atomic"
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
	se.Index = e.eventCounter.Inc()
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
