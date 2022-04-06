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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGoroutinesChecker(t *testing.T) {
	cases := []struct {
		title   string
		test    func(ctl *goroutineTesterControl)
		timeout time.Duration
		fail    bool
	}{
		{
			title: "no goroutines",
			test:  func(ctl *goroutineTesterControl) {},
		},
		{
			title: "fast goroutine",
			test: func(ctl *goroutineTesterControl) {
				ctl.startGoroutine(func() {})
			},
		},
		{
			title: "blocked goroutine",
			test: func(ctl *goroutineTesterControl) {
				ctl.startGoroutine(func() {
					ctl.block()
				})
			},
			timeout: 10 * time.Millisecond,
			fail:    true,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			ctl := newControl()
			defer ctl.cleanup(t)

			goroutines := NewGoroutinesChecker()
			if c.timeout > 0 {
				goroutines.FinalizationTimeout = c.timeout
			}
			c.test(ctl)
			err := goroutines.check()
			if c.fail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// goroutineTesterControl helps keeping track of goroutines started for each test case.
type goroutineTesterControl struct {
	checker GoroutinesChecker
	blocker chan struct{}
}

func newControl() *goroutineTesterControl {
	return &goroutineTesterControl{
		checker: NewGoroutinesChecker(),
		blocker: make(chan struct{}),
	}
}

// startGoroutine ensures that a goroutine is started before continuing.
func (c *goroutineTesterControl) startGoroutine(f func()) {
	started := make(chan struct{})
	go func() {
		started <- struct{}{}
		f()
	}()
	<-started
}

// block blocks forever (being "ever" the life of the test).
func (c *goroutineTesterControl) block() {
	<-c.blocker
}

// cleanup ensures that all started goroutines are finished.
func (c *goroutineTesterControl) cleanup(t *testing.T) {
	close(c.blocker)
	if _, err := c.checker.WaitUntilOriginalCount(); err != nil {
		t.Fatal("goroutines in test cases should be started using startGoroutine")
	}
}
