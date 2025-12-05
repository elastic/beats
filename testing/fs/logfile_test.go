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

package fs

import (
	"sync"
	"testing"
	"time"
)

func TestLogFile(t *testing.T) {
	msg := "it works!"
	lf := NewLogFile(t, "", "")
	if _, err := lf.WriteString(msg + "\n"); err != nil {
		t.Fatalf("cannot write to file: %s", err)
	}

	// Ensure we can find the string we wrote
	lf.LogContains(t, msg)

	// calling FindInLogs will fail because it tracks the offset
	found, err := lf.FindInLogs(msg)
	if err != nil {
		t.Fatalf("cannot search in log file: %s", err)
	}

	if found {
		t.Error("'FindInLogs' must keep track of offset, it should have returned false")
	}

	lf.ResetOffset()
	found, err = lf.FindInLogs(msg)
	if err != nil {
		t.Fatalf("cannot search in log file: %s", err)
	}

	if !found {
		t.Error("offset was reset, 'FindInLogs' must succeed")
	}

	msg2 := "second message"
	wg := sync.WaitGroup{}
	wg.Add(1)

	wgRunning := sync.WaitGroup{}
	wgRunning.Add(1)
	go func() {
		wgRunning.Done()
		lf.WaitLogsContains(t, msg2, 5*time.Second, "did not find msg2")
		wg.Done()
	}()

	// Ensure the goroutine that calls WaitLogsContains is running
	wgRunning.Wait()

	// Write to the file
	if _, err := lf.WriteString(msg2 + "\n"); err != nil {
		t.Fatalf("cannot write to file: %s", err)
	}

	// Ensure the goroutine finishes without failing the tests
	wg.Wait()
	if t.Failed() {
		t.Error("WaitLogsContains should have succeeded")
	}
}
