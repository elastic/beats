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

	"github.com/elastic/elastic-agent-libs/logp"
)

// TestStopWaitsForRunReady proves that Stop does not close the done channel
// until Run has set fb.running to true (i.e., reached waitFinished.Wait).
// This guards against a race where the OTel collector calls Shutdown before
// the beat's Run goroutine has initialised its shutdown-signal machinery.
func TestStopWaitsForRunReady(t *testing.T) {
	fb := &Filebeat{
		done:   make(chan struct{}),
		logger: logp.NewNopLogger(),
	}

	stopDone := make(chan struct{})
	go func() {
		defer close(stopDone)
		fb.Stop()
	}()

	// Give Stop a moment to enter its polling loop.
	time.Sleep(150 * time.Millisecond)

	// done must still be open: Stop is waiting for running to become true.
	select {
	case <-fb.done:
		t.Fatal("Stop closed done before Run set running to true")
	default:
	}

	// Simulate Run() reaching the waitFinished.Wait() call.
	fb.running.Store(true)

	// Stop should close done within the next polling interval (≤100 ms).
	select {
	case <-fb.done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Stop did not close done after running was set to true")
	}

	<-stopDone
}
