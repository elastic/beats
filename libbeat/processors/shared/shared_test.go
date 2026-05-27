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

package shared_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/shared"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// fakeProcessor records Run/Close call counts.
type fakeProcessor struct {
	runCalls   atomic.Int64
	closeCalls atomic.Int64
}

func (f *fakeProcessor) Run(event *beat.Event) (*beat.Event, error) {
	f.runCalls.Add(1)
	return event, nil
}

func (f *fakeProcessor) Close() error   { f.closeCalls.Add(1); return nil }
func (f *fakeProcessor) String() string { return "fake" }

func newFakeConstructor(fp *fakeProcessor) processors.Constructor {
	return func(_ *config.C, _ *logp.Logger) (beat.Processor, error) { return fp, nil }
}

func newEvent(id int) *beat.Event {
	return &beat.Event{Fields: mapstr.M{"id": id}}
}

func TestNew_SameConfigReturnsSameInstance(t *testing.T) {
	var created atomic.Int32
	constructor := shared.New(func(_ *config.C, _ *logp.Logger) (beat.Processor, error) {
		created.Add(1)
		return &fakeProcessor{}, nil
	})
	cfg := config.MustNewConfigFrom(map[string]interface{}{"key": "value"})

	p1, err := constructor(cfg, nil)
	require.NoError(t, err)
	p2, err := constructor(cfg, nil)
	require.NoError(t, err)

	assert.Equal(t, int32(1), created.Load())
	assert.Equal(t, p1, p2)
}

func TestNew_DifferentConfigsReturnDifferentInstances(t *testing.T) {
	var created atomic.Int32
	constructor := shared.New(func(_ *config.C, _ *logp.Logger) (beat.Processor, error) {
		created.Add(1)
		return &fakeProcessor{}, nil
	})

	pA, err := constructor(config.MustNewConfigFrom(map[string]interface{}{"key": "A"}), nil)
	require.NoError(t, err)
	pB, err := constructor(config.MustNewConfigFrom(map[string]interface{}{"key": "B"}), nil)
	require.NoError(t, err)

	assert.Equal(t, int32(2), created.Load())
	assert.NotEqual(t, pA, pB)
}

func TestNew_NilConfigReturnsSameInstance(t *testing.T) {
	var created atomic.Int32
	constructor := shared.New(func(_ *config.C, _ *logp.Logger) (beat.Processor, error) {
		created.Add(1)
		return &fakeProcessor{}, nil
	})

	p1, err := constructor(nil, nil)
	require.NoError(t, err)
	p2, err := constructor(nil, nil)
	require.NoError(t, err)

	assert.Equal(t, int32(1), created.Load())
	assert.Equal(t, p1, p2)
}

func TestNew_NonCloserIsNotWrapped(t *testing.T) {
	type simpleProc struct{ beat.Processor }
	inner := &simpleProc{}
	constructor := shared.New(func(_ *config.C, _ *logp.Logger) (beat.Processor, error) { return inner, nil })

	p, err := constructor(nil, nil)
	require.NoError(t, err)
	assert.Equal(t, p, inner)
}

func TestSharedProcessor_CloseUnderlyingWhenLastUserGone(t *testing.T) {
	tests := []struct {
		users, closesBeforeStop int
		expectClosed            bool
	}{
		{2, 2, true},
		{3, 2, false},
		{3, 3, true},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("users=%d closes=%d", tc.users, tc.closesBeforeStop), func(t *testing.T) {
			fp := &fakeProcessor{}
			constructor := shared.New(newFakeConstructor(fp))
			cfg := config.MustNewConfigFrom(map[string]interface{}{"k": "v"})

			procs := make([]beat.Processor, tc.users)
			for i := range procs {
				p, err := constructor(cfg, nil)
				require.NoError(t, err)
				procs[i] = p
			}
			for i := 0; i < tc.closesBeforeStop; i++ {
				require.NoError(t, processors.Close(procs[i]))
			}

			if tc.expectClosed {
				assert.Equal(t, int64(1), fp.closeCalls.Load(), "expected processor to be closed once")
			} else {
				assert.Equal(t, int64(0), fp.closeCalls.Load(), "didn't expect processor to be closed")
			}
		})
	}
}

func TestNew_ConcurrentSameConfig_NoRace(t *testing.T) {
	const goroutines = 50
	var created atomic.Int32
	constructor := shared.New(func(_ *config.C, _ *logp.Logger) (beat.Processor, error) {
		created.Add(1)
		return &fakeProcessor{}, nil
	})
	cfg := config.MustNewConfigFrom(map[string]interface{}{"concurrent": true})

	results := make([]beat.Processor, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			p, err := constructor(cfg, nil)
			if err != nil {
				t.Errorf("goroutine %d: %v", i, err)
				return
			}
			results[i] = p
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), created.Load())
	for i, p := range results {
		assert.Equalf(t, results[0], p, "goroutine %d got different instance", i)
	}
}

func TestSharedProcessor_ConcurrentRun_NoRace(t *testing.T) {
	const goroutines = 100
	fp := &fakeProcessor{}
	proc, err := shared.New(newFakeConstructor(fp))(nil, nil)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			out, err := proc.Run(newEvent(i))
			if err != nil {
				t.Errorf("goroutine %d: %v", i, err)
				return
			}
			if out.Fields["id"] != i {
				t.Errorf("goroutine %d: id mismatch: got %v", i, out.Fields["id"])
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, int64(goroutines), fp.runCalls.Load())
}

func TestSharedProcessor_ConcurrentClose_NoRace(t *testing.T) {
	const users = 20
	fp := &fakeProcessor{}
	constructor := shared.New(newFakeConstructor(fp))
	cfg := config.MustNewConfigFrom(map[string]interface{}{"cc": true})

	procs := make([]beat.Processor, users)
	for i := range procs {
		p, err := constructor(cfg, nil)
		require.NoError(t, err)
		procs[i] = p
	}

	var wg sync.WaitGroup
	wg.Add(users)
	for i := 0; i < users; i++ {
		i := i
		go func() {
			defer wg.Done()
			if err := processors.Close(procs[i]); err != nil {
				t.Errorf("goroutine %d: %v", i, err)
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, int64(1), fp.closeCalls.Load())
}

func TestSharedProcessor_ConcurrentRunAndClose_NoRace(t *testing.T) {
	const runners, closers = 40, 10
	fp := &fakeProcessor{}
	constructor := shared.New(newFakeConstructor(fp))

	procs := make([]beat.Processor, closers+1)
	for i := range procs {
		p, err := constructor(nil, nil)
		require.NoError(t, err)
		procs[i] = p
	}

	stop := make(chan struct{})

	var runWg sync.WaitGroup
	runWg.Add(runners)
	for i := 0; i < runners; i++ {
		i := i
		go func() {
			defer runWg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_, _ = procs[0].Run(newEvent(i))
				}
			}
		}()
	}

	var closeWg sync.WaitGroup
	closeWg.Add(closers)
	for i := 0; i < closers; i++ {
		i := i
		go func() {
			defer closeWg.Done()
			if err := processors.Close(procs[i+1]); err != nil {
				t.Errorf("closer %d: %v", i, err)
			}
		}()
	}
	closeWg.Wait()
	close(stop)
	runWg.Wait()
}

func TestSharedProcessor_RunEventIsolation(t *testing.T) {
	proc, err := shared.New(func(_ *config.C, _ *logp.Logger) (beat.Processor, error) {
		return &fakeProcessor{}, nil
	})(nil, nil)
	require.NoError(t, err)

	e1, err := proc.Run(newEvent(1))
	require.NoError(t, err)
	e2, err := proc.Run(newEvent(2))
	require.NoError(t, err)

	e1.Fields["injected"] = "mutation"
	assert.NotContains(t, e2.Fields, "injected")
	assert.Equal(t, 2, e2.Fields["id"])
}

func TestSafeProcessorWithSafe(t *testing.T) {
	fp := &fakeProcessor{}
	safeProcConstructor := processors.SafeWrap(shared.New(newFakeConstructor(fp)))
	proc1, err := safeProcConstructor(nil, nil)
	require.NoError(t, err)
	proc2, err := safeProcConstructor(nil, nil)
	require.NoError(t, err)

	require.NoError(t, processors.Close(proc1))
	// processors should not be closed
	require.Zero(t, fp.closeCalls.Load())

	// double-close on first instance
	require.NoError(t, processors.Close(proc1))
	// processors should not be closed, yet
	require.Zero(t, fp.closeCalls.Load())

	require.NoError(t, processors.Close(proc2))
	// processors should be closed now.
	require.Equal(t, int64(1), fp.closeCalls.Load())
}
