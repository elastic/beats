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

package beater

import (
	"github.com/elastic/elastic-agent-libs/logp"
)

type signalWait struct {
	count   int // number of potential 'alive' signals
	signals chan struct{}
	ch      <-chan struct{}
}

type signaler func()

func newSignalWait() *signalWait {
	return &signalWait{
		signals: make(chan struct{}, 1),
	}
}

func (s *signalWait) Wait() {
	if s.count == 0 {
		return
	}

	select {
	case <-s.ch:
		// A closed channel stays readable. Drop it so later Wait
		// calls still block until another registered signal fires.
		s.ch = nil
	case <-s.signals:
	}
	s.count--
}

func (s *signalWait) Add(fn signaler) {
	s.count++
	go func() {
		fn()
		var v struct{}
		s.signals <- v
	}()
}

func (s *signalWait) AddChan(c <-chan struct{}) {
	if s.ch == nil {
		s.ch = c // first channel: Wait() selects on this
		s.count++
		return
	}
	s.Add(waitChannel(c)) // extra channels: old forwarder goroutine
}

func waitChannel(c <-chan struct{}) signaler {
	return func() { <-c }
}

func withLog(s signaler, msg string, logger *logp.Logger) signaler {
	return func() {
		s()
		logger.Infof("%v", msg)
	}
}
