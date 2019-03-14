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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWait(t *testing.T) {
	d1 := 100 * time.Millisecond
	d2 := 200 * time.Millisecond

	t.Run("Allow to wait for a period and initial time", func(t *testing.T) {
		waiter := NewPeriodicTimer(Const(d1), Const(d2))
		waiter.Start()
		defer waiter.Stop()

		start := time.Now()
		end := <-waiter.Wait()

		assert.True(t, end.Sub(start) >= d1)

		start = time.Now()
		end = <-waiter.Wait()
		assert.True(t, end.Sub(start) >= d2)
	})

	t.Run("Allow to stop and start a timer", func(t *testing.T) {
		waiter := NewPeriodicTimer(Const(d1), Const(d2))
		waiter.Start()
		if waiter.Stop() {
			<-waiter.Wait()
		}

		waiter.Start()
		defer waiter.Stop()

		start := time.Now()
		end := <-waiter.Wait()

		assert.True(t, end.Sub(start) >= d1)

		start = time.Now()
		end = <-waiter.Wait()
		assert.True(t, end.Sub(start) >= d2)
	})

	t.Run("Allow to reset a timer", func(t *testing.T) {
		waiter := NewPeriodicTimer(Const(d1), Const(d2))
		waiter.Start()
		defer waiter.Stop()

		start := time.Now()
		end := <-waiter.Wait()

		assert.True(t, end.Sub(start) >= d1)
		d3 := 400 * time.Millisecond
		waiter.Reset(d3)

		start = time.Now()
		end = <-waiter.Wait()
		assert.True(t, end.Sub(start) >= d3)
	})
}
