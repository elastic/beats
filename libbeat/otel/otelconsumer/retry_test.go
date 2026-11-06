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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetryBackoff(t *testing.T) {
	// Each step is a failure, after the current back-off has ended when ended is
	// set, or a success when reset is set. A failure must start a back-off when
	// wantStarted and wait within [window/2, window).
	steps := []struct {
		ended       bool
		reset       bool
		window      time.Duration
		wantStarted bool
	}{
		{window: 2 * time.Second, wantStarted: true},
		{window: 2 * time.Second},
		{ended: true, window: 4 * time.Second, wantStarted: true},
		{window: 4 * time.Second},
		{ended: true, window: 8 * time.Second, wantStarted: true},
		{ended: true, window: 8 * time.Second, wantStarted: true},
		{reset: true},
		{window: 2 * time.Second, wantStarted: true},
	}

	b := newRetryBackoff(time.Second, 8*time.Second)
	for i, s := range steps {
		if s.reset {
			b.reset()
			continue
		}
		if s.ended {
			// End the back-off instead of sleeping until it ends.
			b.retryAt = time.Now()
		}
		wait, started := b.next()
		assert.Equalf(t, s.wantStarted, started, "step %d: started", i)
		assert.Lessf(t, wait, s.window, "step %d: wait", i)
		if started {
			assert.GreaterOrEqualf(t, wait, s.window/2, "step %d: wait", i)
		}
	}
}
