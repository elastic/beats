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

package otelconsumer

import (
	"math/rand/v2"
	"sync"
	"time"
)

// retryBackoff is an equal-jitter back-off shared by concurrent Publish calls.
type retryBackoff struct {
	init time.Duration
	max  time.Duration

	mu       sync.Mutex
	duration time.Duration // waits are drawn from [duration/2, duration)
	renew    time.Time     // failures from then on open a new window
}

func newRetryBackoff(init, max time.Duration) *retryBackoff {
	return &retryBackoff{init: init, max: max, duration: init}
}

// next records a failure and returns how long to wait before retrying.
func (b *retryBackoff) next() (wait time.Duration, newWindow bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	if !now.Before(b.renew) {
		b.duration = min(2*b.duration, b.max)
		// Retries from this window wait at least duration/2.
		b.renew = now.Add(b.duration / 2)
		newWindow = true
	}
	half := b.duration / 2
	return half + rand.N(half), newWindow
}

func (b *retryBackoff) reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.duration = b.init
	b.renew = time.Time{}
}
