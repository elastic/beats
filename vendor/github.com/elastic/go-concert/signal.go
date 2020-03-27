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

package concert

import (
	"context"
	"sync"
)

// OnceSignaler provides a channel that can only be closed once.
// In addition to the channel one can install callbacks to be executed
// if the signal is triggered.
// Once triggered all further close attempts will be ignored.
//
// The zero value is not valid. NewOnceSignaler must be used to create an
// instance backed by a channel.
type OnceSignaler struct {
	once sync.Once
	mu   sync.Mutex
	ch   chan struct{}
	fn   func()
}

// Canceled is the error returned when the signaler has been triggered.
var Canceled = context.Canceled

var closedChan = func() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}()

// ClosedChan returns a closed read only channel.
func ClosedChan() <-chan struct{} {
	return closedChan
}

// NewOnceSignaler create a new OnceSignaler.
func NewOnceSignaler() *OnceSignaler {
	return &OnceSignaler{
		ch: make(chan struct{}),
	}
}

// OnSignal installs a callback that will be executed if the signal is
// triggered. The callback will be called immediately if the signal has already
// been triggered.
func (s *OnceSignaler) OnSignal(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.ch:
		fn()
	default:
		s.add(fn)
	}
}

func (s *OnceSignaler) add(fn func()) {
	old := s.fn
	if old == nil {
		s.fn = fn
	} else {
		s.fn = func() {
			old()
			fn()
		}
	}
}

// Trigger triggers the signal, closing the channel returned by Done and
// calling all callbacks.
func (s *OnceSignaler) Trigger() {
	s.once.Do(func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		close(s.ch)
		if s.fn != nil {
			s.fn()
			s.fn = nil
		}
	})
}

// Done returns a channel one can listen on to check the the signaler has already been triggered.
func (s *OnceSignaler) Done() <-chan struct{} {
	return s.ch
}

// Err reports an Canceled event if the signaler has been triggered already
func (s *OnceSignaler) Err() error {
	select {
	case <-s.ch:
		return Canceled
	default:
		return nil
	}
}
