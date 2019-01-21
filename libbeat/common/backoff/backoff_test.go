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

package backoff

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type factory func(<-chan struct{}) Backoff

func TestBackoff(t *testing.T) {
	t.Run("test close channel", testCloseChannel)
	t.Run("test unblock after some time", testUnblockAfterInit)
}

func testCloseChannel(t *testing.T) {
	init := 2 * time.Second
	max := 5 * time.Minute

	tests := map[string]factory{
		"ExpBackoff": func(done <-chan struct{}) Backoff {
			return NewExpBackoff(done, init, max)
		},
		"EqualJitterBackoff": func(done <-chan struct{}) Backoff {
			return NewEqualJitterBackoff(done, init, max)
		},
	}

	for name, f := range tests {
		t.Run(name, func(t *testing.T) {
			c := make(chan struct{})
			b := f(c)
			close(c)
			assert.False(t, b.Wait())
		})
	}
}

func testUnblockAfterInit(t *testing.T) {
	init := 1 * time.Second
	max := 5 * time.Minute

	tests := map[string]factory{
		"ExpBackoff": func(done <-chan struct{}) Backoff {
			return NewExpBackoff(done, init, max)
		},
		"EqualJitterBackoff": func(done <-chan struct{}) Backoff {
			return NewEqualJitterBackoff(done, init, max)
		},
	}

	for name, f := range tests {
		t.Run(name, func(t *testing.T) {
			c := make(chan struct{})
			defer close(c)

			b := f(c)

			startedAt := time.Now()
			assert.True(t, WaitOnError(b, errors.New("bad bad")))
			assert.True(t, time.Now().Sub(startedAt) >= init)
		})
	}
}
