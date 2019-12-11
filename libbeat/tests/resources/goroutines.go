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

const defaultFinalizationTimeout = 5 * time.Second

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
	err := c.check(t)
	if err != nil {
		t.Error(err)
	}
}

func (c GoroutinesChecker) check(t testing.TB) error {
	timeout := time.Now().Add(c.FinalizationTimeout)
	var after int
	for time.Now().Before(timeout) {
		after = runtime.NumGoroutine()
		if after <= c.before {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	profile := pprof.Lookup("goroutine")
	profile.WriteTo(os.Stdout, 2)
	return fmt.Errorf("Possible goroutines leak, before: %d, after: %d", c.before, after)
}

// CallAndCheckGoroutines calls a function and checks if it has increased
// the number of goroutines
func CallAndCheckGoroutines(t testing.TB, f func()) {
	t.Helper()
	c := NewGoroutinesChecker()
	f()
	c.Check(t)
}
