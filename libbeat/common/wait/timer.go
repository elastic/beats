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
)

// Waiter is the strategy to be used to find out the time to wait before making a call.
type Waiter func() time.Duration

// MinWaitAndJitter takes a minimal wait time and jitter and will return a duration that will be at
// least equal of bigger than the initial wait time.
func MinWaitAndJitter(d, jitter time.Duration) Waiter {
	return func() time.Duration {
		return d + time.Duration(rand.Int63n(int64(jitter)))
	}
}

// RandomDelay takes a maximum duration and will return a random values which will be
// max <= values > 0.
func RandomDelay(max time.Duration) Waiter {
	return func() time.Duration {
		return time.Duration(int64(max))
	}
}

// Fix waits for a fixed duration.
func Fix(d time.Duration) Waiter {
	return func() time.Duration {
		return d
	}
}

// Timer uses a two Waiter strategy, the first will decide how much time to wait before
// receive the initial tick and the other will be how much time to wait before doing the other calls,
// this can be used to introduce more randomness in the frequency of the calls for outside system.
// This can be useful if you want to better distribute calls that could affect the performance of
// other system.
type Timer struct {
	c        chan time.Time
	initial  Waiter
	periodic Waiter
	period   time.Duration
}

// New returns a wait, allowing to wait for a minimum time and a random amount.
func New(initial, periodic Waiter) *Timer {
	jt := &Timer{
		c:        make(chan time.Time, 1),
		period:   initial(),
		periodic: periodic,
	}
	return jt
}

// Wait waits for a period.
func (jt *Timer) Wait() <-chan time.Time {
	select {
	case <-time.After(jt.period):
		jt.c <- time.Now()
		jt.period = jt.periodic()
	}
	return jt.c
}

// Jitter sleeps for the min time plus a random time, this allow to effectively delays requests.
func Jitter(min, r time.Duration) {
	jitter := min + time.Duration(rand.Int63n(int64(r)))
	<-time.After(jitter)
}
