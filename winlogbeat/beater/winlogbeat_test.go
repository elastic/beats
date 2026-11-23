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

package beater

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStopTracker(t *testing.T) {
	t.Run("all distinct names stop", func(t *testing.T) {
		var calls atomic.Int32
		tr := newStopTracker(2, func() { calls.Add(1) })
		tr.stopped("a")
		assert.Equal(t, int32(0), calls.Load(), "stop called with one log still running")
		tr.stopped("b")
		assert.Equal(t, int32(1), calls.Load(), "stop should be called once all logs stopped")
	})

	t.Run("repeated name counts once", func(t *testing.T) {
		var calls atomic.Int32
		tr := newStopTracker(2, func() { calls.Add(1) })
		tr.stopped("a")
		tr.stopped("a")
		assert.Equal(t, int32(0), calls.Load(), "stop called with one log still running")
	})

	t.Run("stop called exactly once", func(t *testing.T) {
		var calls atomic.Int32
		tr := newStopTracker(1, func() { calls.Add(1) })
		tr.stopped("a")
		tr.stopped("a")
		tr.stopped("a")
		assert.Equal(t, int32(1), calls.Load(), "stop should be called exactly once")
	})

	t.Run("concurrent", func(t *testing.T) {
		var calls atomic.Int32
		tr := newStopTracker(3, func() { calls.Add(1) })
		var wg sync.WaitGroup
		for i := range 3 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				tr.stopped(fmt.Sprint(i))
			}()
		}
		wg.Wait()
		assert.Equal(t, int32(1), calls.Load(), "stop should be called exactly once")
	})
}
