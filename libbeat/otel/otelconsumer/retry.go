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
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/backoff"
)

// retryBackoff is a back-off shared by concurrent Publish calls. Many batches
// are in flight at once and fail together when the output is down, so only the
// first failure starts a back-off and the others retry when it ends. The delay
// escalates only when a failure comes after that, meaning the retries failed.
type retryBackoff struct {
	mu      sync.Mutex
	bo      *backoff.EqualJitterBackoff
	retryAt time.Time // end of the current back-off
}

func newRetryBackoff(init, max time.Duration) *retryBackoff {
	// The constructor returns the Backoff interface, which lacks Next.
	bo, _ := backoff.NewEqualJitterBackoff(init, max).(*backoff.EqualJitterBackoff)
	return &retryBackoff{bo: bo}
}

// next records a failure and returns how long to wait before retrying. started
// reports whether the failure started a new back-off instead of joining one.
func (b *retryBackoff) next() (wait time.Duration, started bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	if now.Before(b.retryAt) {
		return b.retryAt.Sub(now), false
	}
	wait = b.bo.Next()
	b.retryAt = now.Add(wait)
	return wait, true
}

func (b *retryBackoff) reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.bo.Reset()
	b.retryAt = time.Time{}
}
