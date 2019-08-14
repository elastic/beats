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

package op

import (
	"fmt"
	"sync/atomic"
)

type Signaler interface {
	Completed()
	Failed()
	Canceled()
}

// splitSignal guards one output signaler from multiple calls
// by using a simple reference counting scheme. If one Signaler consumer
// reports a Failed event, the Failed event will be send to the guarded Signaler
// once the reference count becomes zero.
//
// Example use cases:
//   - Push signaler to multiple outputers
//   - split data to be send in smaller batches
type splitSignal struct {
	count    int32
	signaler Signaler

	// Flags to compute final state.
	// Use atomic ops to determine final SignalResponse
	canceled uint32
	failed   uint32
}

// compositeSignal combines multiple signalers into one Signaler forwarding an
// event to to all signalers.
type compositeSignal struct {
	signalers []Signaler
}

type cancelableSignal struct {
	canceler *Canceler
	signaler Signaler
}

// SignalCallback converts a function accepting SignalResponse into
// a Signaler.
type SignalCallback func(SignalResponse)

type SignalChannel struct {
	C chan SignalResponse
}

type SignalResponse uint8

const (
	SignalCompleted SignalResponse = iota + 1
	SignalFailed
	SignalCanceled
)

func (f SignalCallback) Completed() {
	f(SignalCompleted)
}

func (f SignalCallback) Failed() {
	f(SignalFailed)
}

func (f SignalCallback) Canceled() {
	f(SignalCanceled)
}

func (code SignalResponse) Apply(s Signaler) {
	if s == nil {
		return
	}

	switch code {
	case SignalCompleted:
		s.Completed()
	case SignalFailed:
		s.Failed()
	case SignalCanceled:
		s.Canceled()
	default:
		panic(fmt.Errorf("Invalid signaler code: %v", code))
	}
}

// NewSplitSignaler creates a new splitSignal if s is not nil.
// If s is nil, nil will be returned. The count is the number of events to be
// received before publishing the final event to the guarded Signaler.
func SplitSignaler(s Signaler, count int) Signaler {
	if s == nil {
		return nil
	}

	return &splitSignal{
		count:    int32(count),
		signaler: s,
	}
}

// Completed signals a Completed event to s.
func (s *splitSignal) Completed() {
	s.onEvent()
}

// Failed signals a Failed event to s.
func (s *splitSignal) Failed() {
	atomic.StoreUint32(&s.failed, 1)
	s.onEvent()
}

func (s *splitSignal) Canceled() {
	atomic.StoreUint32(&s.canceled, 1)
	s.onEvent()
}

func (s *splitSignal) onEvent() {
	res := atomic.AddInt32(&s.count, -1)
	if res == 0 {
		canceled := atomic.LoadUint32(&s.canceled)
		failed := atomic.LoadUint32(&s.failed)

		if canceled == 1 {
			s.signaler.Canceled()
		} else if failed == 1 {
			s.signaler.Failed()
		} else {
			s.signaler.Completed()
		}
	}
}

// NewCompositeSignaler creates a new composite signaler.
func CombineSignalers(signalers ...Signaler) Signaler {
	if len(signalers) == 0 {
		return nil
	}
	return &compositeSignal{signalers}
}

// Completed sends the Completed signal to all signalers.
func (cs *compositeSignal) Completed() {
	for _, s := range cs.signalers {
		if s != nil {
			s.Completed()
		}
	}
}

// Failed sends the Failed signal to all signalers.
func (cs *compositeSignal) Failed() {
	for _, s := range cs.signalers {
		if s != nil {
			s.Failed()
		}
	}
}

// Canceled sends the Completed signal to all signalers.
func (cs *compositeSignal) Canceled() {
	for _, s := range cs.signalers {
		if s != nil {
			s.Canceled()
		}
	}
}

func CancelableSignaler(c *Canceler, s Signaler) Signaler {
	if s == nil {
		return nil
	}
	return &cancelableSignal{canceler: c, signaler: s}
}

func (s *cancelableSignal) Completed() {
	l := &s.canceler.lock

	l.RLock()
	if s.canceler.active {
		defer l.RUnlock()
		s.signaler.Completed()
	} else {
		l.RUnlock()
		s.signaler.Canceled()
	}
}

func (s *cancelableSignal) Failed() {
	l := &s.canceler.lock

	l.RLock()
	if s.canceler.active {
		defer l.RUnlock()
		s.signaler.Failed()
	} else {
		l.RUnlock()
		s.signaler.Canceled()
	}
}

func (s *cancelableSignal) Canceled() {
	s.signaler.Canceled()
}

func NewSignalChannel() *SignalChannel {
	return &SignalChannel{make(chan SignalResponse, 1)}
}

func (s *SignalChannel) Completed() {
	s.C <- SignalCompleted
}

func (s *SignalChannel) Failed() {
	s.C <- SignalFailed
}

func (s *SignalChannel) Canceled() {
	s.C <- SignalCanceled
}

func (s *SignalChannel) Wait() SignalResponse {
	return <-s.C
}
