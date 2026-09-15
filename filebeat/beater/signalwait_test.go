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
	"testing"
	"time"
)

// TestSignalWaitSubsequentWaitBlocks checks that Wait consumes one signal per
// call. After the first registered channel fires, a later Wait must still
// block until another signal fires — it must not return just because the
// first channel is already closed.
func TestSignalWaitSubsequentWaitBlocks(t *testing.T) {
	sw := newSignalWait()

	c1 := make(chan struct{})
	c2 := make(chan struct{})
	sw.AddChan(c1)
	sw.AddChan(c2)

	close(c1)
	sw.Wait()

	second := make(chan struct{})
	go func() {
		sw.Wait()
		close(second)
	}()

	select {
	case <-second:
		t.Fatal("second Wait returned without another signal")
	case <-time.After(100 * time.Millisecond):
	}

	close(c2)

	select {
	case <-second:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("second Wait did not return after the remaining signal fired")
	}
}
