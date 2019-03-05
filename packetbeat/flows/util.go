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

package flows

import (
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/logp"
)

type worker struct {
	wg   sync.WaitGroup
	done chan struct{}
	run  func(*worker)
}

type spool struct {
	pub    Reporter
	events []beat.Event
}

func newWorker(fn func(w *worker)) *worker {
	return &worker{
		done: make(chan struct{}),
		run:  fn,
	}
}

func (w *worker) Start() {
	debugf("start flows worker")
	w.wg.Add(1)
	go func() {
		defer w.finished()
		w.run(w)
	}()
}

func (w *worker) Stop() {
	debugf("stop flows worker")
	close(w.done)
	w.wg.Wait()
	debugf("stopped flows worker")
}

func (w *worker) finished() {
	w.wg.Done()
	logp.Info("flows worker loop stopped")
}

func (w *worker) sleep(d time.Duration) bool {
	select {
	case <-w.done:
		return false
	case <-time.After(d):
		return true
	}
}

func (w *worker) tick(t *time.Ticker) bool {
	select {
	case <-w.done:
		return false
	case <-t.C:
		return true
	}
}

func (w *worker) periodically(tick time.Duration, fn func() error) {
	defer debugf("stop periodic loop")

	ticker := time.NewTicker(tick)
	for {
		cont := w.tick(ticker)
		if !cont {
			return
		}

		err := fn()
		if err != nil {
			return
		}
	}
}

func (s *spool) init(pub Reporter, sz int) {
	s.pub = pub
	s.events = make([]beat.Event, 0, sz)
}

func (s *spool) publish(event beat.Event) {
	s.events = append(s.events, event)
	if len(s.events) == cap(s.events) {
		s.flush()
	}
}

func (s *spool) flush() {
	if len(s.events) == 0 {
		return
	}

	s.pub(s.events)
	s.events = make([]beat.Event, 0, cap(s.events))
}

func gcd(a, b int64) int64 {
	if a < 0 || b < 0 {
		return 0
	}

	switch {
	case a == b:
		return a
	case a == 0:
		return b
	case b == 0:
		return a
	}

	shift := uint(0)
	for (a&1) == 0 && (b&1) == 0 {
		shift++
		a /= 2
		b /= 2
	}

	for (a & 1) == 0 {
		a = a / 2
	}

	// a is always odd
	for {
		for (b & 1) == 0 {
			b = b / 2
		}

		// both a and b are odd. guaranteed b >= a
		if a > b {
			a, b = b, a
		}
		b -= a

		if b == 0 {
			break
		}
	}

	// restore common factors of 2
	return a << shift
}
