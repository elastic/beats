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

//go:build !integration

package input

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// fakeInput is a no-op input.Input whose Stop hook lets tests observe ordering.
type fakeInput struct {
	onStop func()
}

func (f *fakeInput) Run() {}
func (f *fakeInput) Stop() {
	if f.onStop != nil {
		f.onStop()
	}
}
func (f *fakeInput) Wait() {}

type recordingCloser struct {
	closed  bool
	onClose func()
}

func (c *recordingCloser) Close() error {
	c.closed = true
	if c.onClose != nil {
		c.onClose()
	}
	return nil
}

// TestRunnerStopClosesClosersAfterInputStops asserts the channel.InputRunner
// contract: closers run on Stop, and only after the input has stopped.
func TestRunnerStopClosesClosersAfterInputStops(t *testing.T) {
	var mu sync.Mutex
	var order []string
	record := func(s string) {
		mu.Lock()
		defer mu.Unlock()
		order = append(order, s)
	}

	r := &Runner{
		input:  &fakeInput{onStop: func() { record("input.Stop") }},
		done:   make(chan struct{}),
		wg:     &sync.WaitGroup{},
		logger: logptest.NewTestingLogger(t, ""),
	}
	// Large scan frequency so Run blocks on done instead of busy re-scanning.
	r.config.ScanFrequency = time.Hour

	closer := &recordingCloser{onClose: func() { record("closer.Close") }}
	r.AddCloser(closer)

	r.Start()
	r.Stop()

	require.True(t, closer.closed, "registered closer must be closed on Stop")
	require.Equal(t, []string{"input.Stop", "closer.Close"}, order,
		"input must stop (clients drain) before the shared resources are released")
}

// TestRunnerStopWithoutClosersIsSafe ensures Stop still works when nothing was
// registered.
func TestRunnerStopWithoutClosersIsSafe(t *testing.T) {
	r := &Runner{
		input:  &fakeInput{},
		done:   make(chan struct{}),
		wg:     &sync.WaitGroup{},
		logger: logptest.NewTestingLogger(t, ""),
	}
	r.config.ScanFrequency = time.Hour

	r.Start()
	require.NotPanics(t, r.Stop)
}
