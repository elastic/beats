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

package monitors

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSimpleWait(t *testing.T) {
	tests := map[string]struct {
		number int
	}{
		"one wait signals": {
			number: 1,
		},
		"50 wait signals": {
			number: 50,
		},
		"100 wait signals": {
			number: 100,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			done := make(chan bool)

			gl := sync.WaitGroup{}
			gl.Add(1) // Lock routines indefinitely

			addFn := func(s *signalWait) {
				s.Add(func() {
					gl.Wait()
					done <- true // Just in case, shared channel
				})
			}

			signalWait := NewSignalWait()

			for i := 0; i < tt.number; i++ {
				addFn(signalWait)
			}

			go func() {
				signalWait.Wait()
				done <- true
			}()

			wait := time.After(500 * time.Millisecond)
			select {
			case <-done:
				assert.Fail(t, "found early exit signal")
			case <-wait:
			}

			signalWait.Add(func() {})

			wait = time.After(500 * time.Millisecond)
			select {
			case <-done:
			case <-wait:
				assert.Fail(t, "signal did not exit on time")
			}
		})
	}
}

func TestChannelWait(t *testing.T) {
	tests := map[string]struct {
		number int
	}{
		"one wait signals": {
			number: 1,
		},
		"50 wait signals": {
			number: 50,
		},
		"100 wait signals": {
			number: 100,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			done := make(chan bool)
			gl := make(chan struct{})
			signalWait := NewSignalWait()

			for i := 0; i < tt.number; i++ {
				signalWait.AddChan(gl)
			}

			go func() {
				signalWait.Wait()
				done <- true
			}()

			wait := time.After(500 * time.Millisecond)
			select {
			case <-done:
				assert.Fail(t, "found early exit signal")
			case <-wait:
			}

			d := make(chan struct{})
			signalWait.AddChan(d)
			d <- struct{}{}

			wait = time.After(500 * time.Millisecond)
			select {
			case <-done:
			case <-wait:
				assert.Fail(t, "signal did not exit on time")
			}
		})
	}
}

func TestTimeoutWait(t *testing.T) {
	tests := map[string]struct {
		number int
	}{
		"one wait signals": {
			number: 1,
		},
		"50 wait signals": {
			number: 50,
		},
		"100 wait signals": {
			number: 100,
		},
	}

	for name, tt := range tests {

		t.Run(name, func(t *testing.T) {
			done := make(chan bool)

			signalWait := NewSignalWait()

			for i := 0; i < tt.number; i++ {
				signalWait.AddTimer(time.NewTimer(time.Hour))
			}

			go func() {
				signalWait.Wait()
				done <- true
			}()

			wait := time.After(500 * time.Millisecond)
			select {
			case <-done:
				assert.Fail(t, "found early exit signal")
			case <-wait:
			}

			signalWait.AddTimer(time.NewTimer(time.Microsecond))

			wait = time.After(500 * time.Millisecond)
			select {
			case <-done:
			case <-wait:
				assert.Fail(t, "signal did not exit on time")
			}
		})
	}
}
func TestDurationWait(t *testing.T) {
	tests := map[string]struct {
		number int
	}{
		"one wait signals": {
			number: 1,
		},
		"50 wait signals": {
			number: 50,
		},
		"100 wait signals": {
			number: 100,
		},
	}

	for name, tt := range tests {

		t.Run(name, func(t *testing.T) {
			done := make(chan bool)

			signalWait := NewSignalWait()

			for i := 0; i < tt.number; i++ {
				signalWait.AddTimeout(time.Hour)
			}

			go func() {
				signalWait.Wait()
				done <- true
			}()

			wait := time.After(500 * time.Millisecond)
			select {
			case <-done:
				assert.Fail(t, "found early exit signal")
			case <-wait:
			}

			signalWait.AddTimeout(time.Microsecond)

			wait = time.After(500 * time.Millisecond)
			select {
			case <-done:
			case <-wait:
				assert.Fail(t, "signal did not exit on time")
			}
		})
	}
}
