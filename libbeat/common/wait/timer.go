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

package wait

import (
	"math/rand"
	"time"

	"github.com/elastic/beats/libbeat/common/atomic"
)

// Timer represents a timer implementation.
type Timer interface {
	Reset(time.Duration)
	Start()
	Stop()
	Wait() <-chan time.Time
}

// Strategy is the strategy to be used to find out the time to wait before making a call.
type Strategy func() time.Duration

// MinWaitAndJitter takes a minimal wait time and jitter and will return a duration that will be at
// least equal of bigger than the initial wait time.
func MinWaitAndJitter(d, jitter time.Duration) Strategy {
	return func() time.Duration {
		return d + time.Duration(rand.Int63n(int64(jitter)))
	}
}

// RandomDelay takes a maximum duration and will return a random values which will be
// max <= values > 0.
func RandomDelay(max time.Duration) Strategy {
	return func() time.Duration {
		return time.Duration(int64(max))
	}
}

// Const waits for a fixed duration.
func Const(d time.Duration) Strategy {
	return func() time.Duration {
		return d
	}
}

// PeriodicTimer uses a two Waiter strategy, the first will decide how much time to wait before
// receive the initial tick and the other will be how much time to wait before doing the other calls,
// this can be used to introduce more randomness in the frequency of the calls for outside system.
// This can be useful if you want to better distribute calls that could affect the performance of
// other system.
type PeriodicTimer struct {
	c           chan time.Time
	resetOrDone chan time.Duration
	initial     Strategy
	periodic    Strategy
	period      time.Duration
	running     atomic.Bool
}

// NewPeriodicTimer returns a wait, allowing to wait for a minimum time and a random amount.
func NewPeriodicTimer(initial, periodic Strategy) *PeriodicTimer {
	jt := &PeriodicTimer{
		c:           make(chan time.Time),
		resetOrDone: make(chan time.Duration, 1),
		period:      initial(),
		periodic:    periodic,
		running:     atomic.MakeBool(false),
	}
	return jt
}

// Start starts the timer.
func (jt *PeriodicTimer) Start() {
	if !jt.running.Load() {
		jt.running.Store(true)
		go jt.startTimer()
	}
}

func (jt *PeriodicTimer) startTimer() {
	for {
		select {
		case reset, ok := <-jt.resetOrDone:
			if !ok {
				return
			}
			jt.period = reset
		case <-time.After(jt.period):
			jt.c <- time.Now()
			jt.period = jt.periodic()
		}
	}
}

// Wait returns a channel that will receives tick.
func (jt *PeriodicTimer) Wait() <-chan time.Time {
	return jt.c
}

// Reset resets the current timer with the provided duration
// NOTE: it is possible to receive a tick before the reset actually happen.
func (jt *PeriodicTimer) Reset(d time.Duration) {
	jt.resetOrDone <- d
}

// Stop stops the current timer but won't close the channel, this prevent a goroutine to received
// a bad tick. This is the same strategy used by the time.Ticker.
func (jt *PeriodicTimer) Stop() bool {
	if jt.running.Load() {
		close(jt.resetOrDone)
		return true
	}
	return false
}

// Jitter sleeps for the min time plus a random time, this allow to effectively delays requests.
func Jitter(min, r time.Duration) {
	jitter := min + time.Duration(rand.Int63n(int64(r)))
	<-time.After(jitter)
}
