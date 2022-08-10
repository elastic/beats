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

package resources

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"
)

// This is the maximum waiting time for goroutine shutdown.
// If the shutdown happens earlier the waiting time will be lower.
// High maximum waiting time was due to flaky tests on CI workers
const defaultFinalizationTimeout = 35 * time.Second

// GoroutinesChecker keeps the count of goroutines when it was created
// so later it can check if this number has increased
type GoroutinesChecker struct {
	before int

	// FinalizationTimeout is the time to wait till goroutines have finished
	FinalizationTimeout time.Duration
}

// NewGoroutinesChecker creates a new GoroutinesChecker
func NewGoroutinesChecker() GoroutinesChecker {
	return GoroutinesChecker{
		before:              runtime.NumGoroutine(),
		FinalizationTimeout: defaultFinalizationTimeout,
	}
}

// Check if the number of goroutines has increased since the checker
// was created
func (c GoroutinesChecker) Check(t testing.TB) {
	t.Helper()
	err := c.check()
	if err != nil {
		dumpGoroutines()
		t.Error(err)
	}
}

func dumpGoroutines() {
	profile := pprof.Lookup("goroutine")
	profile.WriteTo(os.Stdout, 2)
}

func (c GoroutinesChecker) check() error {
	after, err := c.WaitUntilOriginalCount()
	if err == ErrTimeout {
		return fmt.Errorf("possible goroutines leak, before: %d, after: %d", c.before, after)
	}
	return err
}

// CallAndCheckGoroutines calls a function and checks if it has increased
// the number of goroutines
func CallAndCheckGoroutines(t testing.TB, f func()) {
	t.Helper()
	c := NewGoroutinesChecker()
	f()
	c.Check(t)
}

// ErrTimeout is the error returned when WaitUntilOriginalCount timeouts.
var ErrTimeout = fmt.Errorf("timeout waiting for finalization of goroutines")

// WaitUntilOriginalCount waits until the original number of goroutines are
// present before we created the resource checker.
// It returns the number of goroutines after the check and a timeout error
// in case the wait has expired.
func (c GoroutinesChecker) WaitUntilOriginalCount() (int, error) {
	timeout := time.Now().Add(c.FinalizationTimeout)

	var after int
	for time.Now().Before(timeout) {
		after = runtime.NumGoroutine()
		if after <= c.before {
			return after, nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return after, ErrTimeout
}

// WaitUntilIncreased waits till the number of goroutines is n plus the number
// before creating the checker.
func (c *GoroutinesChecker) WaitUntilIncreased(n int) {
	for runtime.NumGoroutine() < c.before+n {
		time.Sleep(10 * time.Millisecond)
	}
}
