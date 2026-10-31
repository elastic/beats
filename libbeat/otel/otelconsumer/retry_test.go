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
	// Each step is a failure, after the current window has elapsed when elapsed
	// is set. It must open a new window when wantNewWindow and wait within
	// [window/2, window).
	steps := []struct {
		elapsed       bool
		window        time.Duration
		wantNewWindow bool
	}{
		{window: 2 * time.Second, wantNewWindow: true},
		{window: 2 * time.Second},
		{elapsed: true, window: 4 * time.Second, wantNewWindow: true},
		{window: 4 * time.Second},
		{elapsed: true, window: 8 * time.Second, wantNewWindow: true},
		{elapsed: true, window: 8 * time.Second, wantNewWindow: true},
	}

	b := newRetryBackoff(time.Second, 8*time.Second)
	for i, s := range steps {
		if s.elapsed {
			// Move the window boundary instead of sleeping until it.
			b.renew = time.Now()
		}
		wait, newWindow := b.next()
		assert.Equalf(t, s.wantNewWindow, newWindow, "step %d: new window", i)
		assert.GreaterOrEqualf(t, wait, s.window/2, "step %d: wait", i)
		assert.Lessf(t, wait, s.window, "step %d: wait", i)
	}
}
