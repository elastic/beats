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

package input_logfile

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// TestEngineLifecycle covers the reference counting that decides when the
// shared waker runs. These tests use the process-global engine, so they must
// leave the reference count at zero.
func TestEngineLifecycle(t *testing.T) {
	log := logptest.NewTestingLogger(t, "")

	t.Run("the engine survives until the last input leaves", func(t *testing.T) {
		requireNoEngine(t)

		first, releaseFirst := acquireEngine(log)
		second, releaseSecond := acquireEngine(log)
		require.Same(t, first, second, "inputs in one process share a single engine")

		releaseFirst()
		require.True(t, engineIsLive(), "the engine must outlive an input while another still uses it")

		releaseSecond()
		requireNoEngine(t)
	})

	t.Run("release is idempotent", func(t *testing.T) {
		requireNoEngine(t)

		_, releaseFirst := acquireEngine(log)
		_, releaseSecond := acquireEngine(log)

		releaseFirst()
		releaseFirst() // a second call must not drop the other input's reference
		require.True(t, engineIsLive(), "a repeated release must not tear the engine down")

		releaseSecond()
		requireNoEngine(t)
	})

	// Covers every filestream input being reconfigured away and one starting
	// again a moment later: the replacement must be usable, not a stopped engine
	// whose waker exits immediately.
	t.Run("an input starting after a full stop gets a running engine", func(t *testing.T) {
		requireNoEngine(t)

		first, releaseFirst := acquireEngine(log)
		releaseFirst()
		requireNoEngine(t)

		second, releaseSecond := acquireEngine(log)
		defer releaseSecond()

		require.NotSame(t, first, second, "a new engine must be built after the last release")

		select {
		case <-second.done:
			t.Fatal("a newly acquired engine must not already be stopped")
		default:
		}

		select {
		case <-first.done:
		default:
			t.Fatal("the engine released by the last input must have been stopped")
		}
	})
}

// TestEngineHandOver covers the window releaseEngine exists to make safe: the
// last input has left and its engine is being stopped outside the shared lock,
// while another input starts. Inputs come and go on every config reload, and
// getting this wrong is quiet — the starting input would hold an engine whose
// waker has exited, so its files would simply stop being read.
//
// Pairs of inputs racing each other reach the window readily; asserting the
// invariant (acquireEngine never returns a stopped engine) rather than trying to
// observe a particular instant is what makes it catchable. A releaseEngine that
// stops the engine before detaching it fails on the first iteration.
func TestEngineHandOver(t *testing.T) {
	log := logptest.NewTestingLogger(t, "")
	requireNoEngine(t)

	for i := range 20000 {
		var wg sync.WaitGroup
		start := make(chan struct{})
		for range 2 {
			wg.Go(func() {
				<-start
				engine, release := acquireEngine(log)
				defer release()

				select {
				case <-engine.done:
					t.Errorf("iteration %d: an input was handed an engine that had been stopped", i)
				default:
				}
			})
		}
		close(start)
		wg.Wait()
	}

	requireNoEngine(t)
}

// engineIsLive reports whether a shared engine is currently registered.
func engineIsLive() bool {
	sharedEngine.Lock()
	defer sharedEngine.Unlock()
	return sharedEngine.engine != nil
}

// requireNoEngine asserts the shared engine has been torn down, so a test that
// leaks a reference fails where it leaked rather than in an unrelated test.
func requireNoEngine(t *testing.T) {
	t.Helper()

	sharedEngine.Lock()
	defer sharedEngine.Unlock()
	require.Nil(t, sharedEngine.engine, "no engine must be registered")
	require.Zero(t, sharedEngine.refs, "no references must be outstanding")
}
