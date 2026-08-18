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

package processors

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/elastic/beats/v9/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/paths"
)

var mockEvent = &beat.Event{}

type mockProcessor struct {
	runCount int
}

func (p *mockProcessor) Run(event *beat.Event) (*beat.Event, error) {
	p.runCount++
	return mockEvent, nil
}

func (p *mockProcessor) String() string {
	return "mock-processor"
}

type mockCloserProcessor struct {
	mockProcessor
	closeCount int
}

func (p *mockCloserProcessor) Close() error {
	p.closeCount++
	return nil
}

type concurrentMockCloserProcessor struct {
	closed atomic.Bool
	runs   atomic.Int32
	closes atomic.Int32
}

func (p *concurrentMockCloserProcessor) Run(event *beat.Event) (*beat.Event, error) {
	if p.closed.Load() {
		return nil, errors.New("processor is closed")
	}
	p.runs.Add(1)
	return event, nil
}

func (p *concurrentMockCloserProcessor) Close() error {
	p.closes.Add(1)
	if p.closed.Swap(true) {
		return errors.New("processor closed twice")
	}
	return nil
}

func (p *concurrentMockCloserProcessor) String() string {
	return "concurrent-mock-closer"
}

func newMockCloserConstructor() (Constructor, *mockCloserProcessor) {
	p := mockCloserProcessor{}
	constructor := func(config *config.C, _ *logp.Logger) (beat.Processor, error) {
		return &p, nil
	}
	return constructor, &p
}

type mockPathSetterCloserProcessor struct {
	mockCloserProcessor
	setPathsCount int
}

func (p *mockPathSetterCloserProcessor) SetPaths(*paths.Path) error {
	p.setPathsCount++
	return nil
}

func newMockPathSetterCloserProcessor() (Constructor, *mockPathSetterCloserProcessor) {
	p := mockPathSetterCloserProcessor{}
	constructor := func(config *config.C, _ *logp.Logger) (beat.Processor, error) { return &p, nil }
	return constructor, &p
}

type mockPathSetterProcessor struct {
	mockProcessor
	setPathsCount int
}

func (p *mockPathSetterProcessor) SetPaths(*paths.Path) error {
	p.setPathsCount++
	return nil
}

func newMockPathSetterProcessor() (Constructor, *mockPathSetterProcessor) {
	p := mockPathSetterProcessor{}
	constructor := func(config *config.C, _ *logp.Logger) (beat.Processor, error) {
		return &p, nil
	}
	return constructor, &p
}

type mockUnshareableProcessor struct {
	mockProcessor
}

func (p *mockUnshareableProcessor) Unshareable() {}

type mockPdataProcessor struct {
	mockProcessor
	pdataCount int
}

func (p *mockPdataProcessor) RunPdata(pcommon.Map) (bool, error) {
	p.pdataCount++
	return false, nil
}

func mockConstructor(config *config.C, log *logp.Logger) (beat.Processor, error) {
	return &mockProcessor{}, nil
}

func mockCloserConstructor(config *config.C, log *logp.Logger) (beat.Processor, error) {
	return &mockCloserProcessor{}, nil
}

func TestSafeWrap(t *testing.T) {
	t.Run("shares a non-closer processor", func(t *testing.T) {
		nonCloser := mockConstructor
		wrappedNonCloser := SafeWrap("shared non-closer processor", nonCloser)
		wp, err := wrappedNonCloser(nil, nil)
		require.NoError(t, err)
		assert.IsType(t, &sharedHandle{}, wp)
		assert.Implements(t, (*Closer)(nil), wp)
		require.NoError(t, Close(wp))
	})

	t.Run("wraps a closer processor", func(t *testing.T) {
		closer := mockCloserConstructor
		wrappedCloser := SafeWrap("closer processor", closer)
		wcp, err := wrappedCloser(nil, nil)
		require.NoError(t, err)
		assert.IsType(t, &sharedHandle{}, wcp)
		assert.Implements(t, (*Closer)(nil), wcp)
		require.NoError(t, Close(wcp))
	})

	t.Run("wraps a closer path-setter processor without sharing", func(t *testing.T) {
		cons, _ := newMockPathSetterCloserProcessor()
		wrapped := SafeWrap("closer path-setter processor", cons)
		wp, err := wrapped(nil, nil)
		require.NoError(t, err)
		assert.IsType(t, &safeProcessorWithClose{}, wp)
		assert.Implements(t, (*Closer)(nil), wp)
	})
}

func TestSafeProcessor(t *testing.T) {
	cons, p := newMockCloserConstructor()
	var (
		sp  beat.Processor
		err error
	)
	t.Run("creates a wrapped processor", func(t *testing.T) {
		sw := SafeWrap("", cons)
		sp, err = sw(nil, nil)
		require.NoError(t, err)
	})

	t.Run("propagates Run to a processor", func(t *testing.T) {
		assert.Equal(t, 0, p.runCount)

		e, err := sp.Run(nil)
		assert.NoError(t, err)
		assert.Equal(t, e, mockEvent)
		e, err = sp.Run(nil)
		assert.NoError(t, err)
		assert.Equal(t, e, mockEvent)

		assert.Equal(t, 2, p.runCount)
	})

	t.Run("propagates Close to a processor only once", func(t *testing.T) {
		assert.Equal(t, 0, p.closeCount)

		err := Close(sp)
		assert.NoError(t, err)
		err = Close(sp)
		assert.NoError(t, err)

		assert.Equal(t, 1, p.closeCount)
	})

	t.Run("does not propagate Run when closed", func(t *testing.T) {
		assert.Equal(t, 2, p.runCount) // still 2 from the previous test case
		e, err := sp.Run(nil)
		assert.Nil(t, e)
		assert.ErrorIs(t, err, ErrClosed)
		assert.Equal(t, 2, p.runCount)
	})
}

func TestSafeProcessorSetPathsClose(t *testing.T) {
	cons, p := newMockPathSetterCloserProcessor()
	var (
		bp  beat.Processor
		sp  PathSetter
		err error
	)
	t.Run("creates a wrapped processor", func(t *testing.T) {
		sw := SafeWrap("", cons)
		bp, err = sw(nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 0, p.setPathsCount)
	})

	t.Run("does not run before SetPaths is called", func(t *testing.T) {
		assert.Equal(t, 0, p.runCount)
		e, err := bp.Run(nil)
		assert.Nil(t, e)
		assert.ErrorIs(t, err, ErrPathsNotSet)
		assert.Equal(t, 0, p.runCount)
	})

	t.Run("sets paths", func(t *testing.T) {
		assert.Equal(t, 0, p.setPathsCount)
		require.Implements(t, (*PathSetter)(nil), bp)
		var ok bool
		sp, ok = bp.(PathSetter)
		require.True(t, ok)
		require.NotNil(t, sp)
		testPaths := &paths.Path{}
		err = sp.SetPaths(testPaths)
		assert.NoError(t, err)
		assert.Equal(t, 1, p.setPathsCount)

		// set paths again with the SAME pointer (idempotent for global processors)
		err = sp.SetPaths(testPaths)
		assert.NoError(t, err)
		assert.Equal(t, 1, p.setPathsCount)

		// set paths again with a DIFFERENT pointer (should error)
		err = sp.SetPaths(&paths.Path{})
		assert.ErrorIs(t, err, ErrPathsAlreadySet)
		assert.Equal(t, 1, p.setPathsCount)
	})

	t.Run("propagates Run to a processor", func(t *testing.T) {
		assert.Equal(t, 0, p.runCount)

		e, err := bp.Run(nil)
		assert.NoError(t, err)
		assert.Equal(t, e, mockEvent)
		e, err = bp.Run(nil)
		assert.NoError(t, err)
		assert.Equal(t, e, mockEvent)

		assert.Equal(t, 2, p.runCount)
	})

	t.Run("propagates Close to a processor only once", func(t *testing.T) {
		assert.Equal(t, 0, p.closeCount)

		err := Close(bp)
		assert.NoError(t, err)
		err = Close(bp)
		assert.NoError(t, err)

		assert.Equal(t, 1, p.closeCount)
	})

	t.Run("does not propagate Run when closed", func(t *testing.T) {
		assert.Equal(t, 2, p.runCount) // still 2 from the previous test case
		e, err := bp.Run(nil)
		assert.Nil(t, e)
		assert.ErrorIs(t, err, ErrClosed)
		assert.Equal(t, 2, p.runCount)
	})

	t.Run("does not set paths when closed", func(t *testing.T) {
		err = sp.SetPaths(&paths.Path{})
		assert.ErrorIs(t, err, ErrSetPathsOnClosed)
		assert.Equal(t, 1, p.setPathsCount)
	})
}

func TestSafeWrapSharedInstanceByNameAndHash(t *testing.T) {
	cons, p := newMockCloserConstructor()
	constructions := 0
	counting := func(cfg *config.C, log *logp.Logger) (beat.Processor, error) {
		constructions++
		return cons(cfg, log)
	}
	sw := SafeWrap("test-shared-instance", counting)

	proc1, err := sw(nil, nil)
	require.NoError(t, err, "first SafeWrap call should succeed")

	proc2, err := sw(nil, nil)
	require.NoError(t, err, "second SafeWrap call should succeed")

	assert.Equal(t, 1, constructions, "same name+config must construct the underlying processor once")
	assert.NotSame(t, proc1, proc2, "each call returns its own handle to the shared instance")

	_, err = proc1.Run(nil)
	require.NoError(t, err, "Run via proc1 should succeed")
	assert.Equal(t, 1, p.runCount, "run should be reflected in the underlying mock")

	require.NoError(t, Close(proc1), "first Close should not error")
	assert.Equal(t, 0, p.closeCount, "underlying processor should not be closed while a ref remains")

	require.NoError(t, Close(proc1), "duplicate Close of the same handle should not error")
	assert.Equal(t, 0, p.closeCount, "duplicate Close must not release another handle's reference")
	_, err = proc2.Run(nil)
	require.NoError(t, err, "proc2 must remain usable after proc1 was closed twice")

	require.NoError(t, Close(proc2), "second Close should not error")
	assert.Equal(t, 1, p.closeCount, "underlying processor should be closed once all refs are released")
}

func TestSafeWrapDifferentNamesNotShared(t *testing.T) {
	cons1, p1 := newMockCloserConstructor()
	cons2, p2 := newMockCloserConstructor()

	proc1, err := SafeWrap("test-name-a", cons1)(nil, nil)
	require.NoError(t, err, "SafeWrap for name-a should succeed")

	proc2, err := SafeWrap("test-name-b", cons2)(nil, nil)
	require.NoError(t, err, "SafeWrap for name-b should succeed")

	assert.NotSame(t, proc1, proc2, "different names must produce separate processor instances")

	_, err = proc1.Run(nil)
	require.NoError(t, err, "Run on proc1 should succeed")
	assert.Equal(t, 1, p1.runCount, "run should only increment p1.runCount")
	assert.Equal(t, 0, p2.runCount, "p2.runCount must remain 0")

	require.NoError(t, Close(proc1))
	require.NoError(t, Close(proc2))
	assert.Equal(t, 1, p1.closeCount, "p1 should be closed exactly once")
	assert.Equal(t, 1, p2.closeCount, "p2 should be closed exactly once")
}

func TestSafeWrapRefCountingPreventsEarlyClose(t *testing.T) {
	cons, p := newMockCloserConstructor()
	sw := SafeWrap("test-refcount", cons)

	proc1, err := sw(nil, nil)
	require.NoError(t, err)
	proc2, err := sw(nil, nil)
	require.NoError(t, err)
	proc3, err := sw(nil, nil)
	require.NoError(t, err)

	require.NoError(t, Close(proc1))
	assert.Equal(t, 0, p.closeCount, "should not close after first of three Close calls")

	require.NoError(t, Close(proc2))
	assert.Equal(t, 0, p.closeCount, "should not close after second of three Close calls")

	require.NoError(t, Close(proc3))
	assert.Equal(t, 1, p.closeCount, "should close exactly once after last ref is released")
}

func TestSafeWrapConcurrentConstructionReservesReferences(t *testing.T) {
	const (
		name    = "test-concurrent-construction-reserves-references"
		callers = 8
	)

	started := make(chan struct{})
	release := make(chan struct{})
	creatorClosed := make(chan struct{})
	underlying := &concurrentMockCloserProcessor{}
	var constructions atomic.Int32
	cons := func(*config.C, *logp.Logger) (beat.Processor, error) {
		if constructions.Add(1) == 1 {
			close(started)
		}
		<-release
		return underlying, nil
	}
	sw := SafeWrap(name, cons)
	logger := logptest.NewTestingLogger(t, "")

	type result struct {
		constructErr error
		runErr       error
		closeErr     error
	}
	results := make(chan result, callers)
	var wg sync.WaitGroup
	wg.Go(func() {
		defer close(creatorClosed)
		p, err := sw(nil, logger)
		var closeErr error
		if err == nil {
			closeErr = Close(p)
		}
		results <- result{constructErr: err, closeErr: closeErr}
	})
	<-started
	for range callers - 1 {
		wg.Go(func() {
			p, err := sw(nil, logger)
			if err != nil {
				results <- result{constructErr: err}
				return
			}
			<-creatorClosed
			_, runErr := p.Run(nil)
			results <- result{runErr: runErr, closeErr: Close(p)}
		})
	}

	allWaiting := waitForSharedReferences(t, name, callers)
	close(release)
	wg.Wait()

	assert.True(t, allWaiting, "all callers must join the in-flight construction")
	assert.Equal(t, int32(1), constructions.Load(),
		"the creator closing before waiters run must not trigger another construction")
	for range callers {
		result := <-results
		assert.NoError(t, result.constructErr, "every caller must receive a handle")
		assert.NoError(t, result.runErr, "waiters must retain a live reference after the creator closes")
		assert.NoError(t, result.closeErr, "every handle must close cleanly")
	}
	assert.Equal(t, int32(callers-1), underlying.runs.Load(), "every waiter must run once")
	assert.Equal(t, int32(1), underlying.closes.Load(), "the underlying processor must close exactly once")
}

func TestSafeWrapConcurrentUnshareableConstruction(t *testing.T) {
	const (
		name    = "test-concurrent-unshareable-construction"
		callers = 8
	)

	started := make(chan struct{})
	release := make(chan struct{})
	var constructions atomic.Int32
	cons := func(*config.C, *logp.Logger) (beat.Processor, error) {
		if constructions.Add(1) == 1 {
			close(started)
			<-release
		}
		return &mockUnshareableProcessor{}, nil
	}
	sw := SafeWrap(name, cons)
	logger := logptest.NewTestingLogger(t, "")

	type result struct {
		processor beat.Processor
		err       error
	}
	results := make(chan result, callers)
	call := func() {
		p, err := sw(nil, logger)
		results <- result{processor: p, err: err}
	}

	var wg sync.WaitGroup
	wg.Go(call)
	<-started
	for range callers - 1 {
		wg.Go(call)
	}

	allWaiting := waitForSharedReferences(t, name, callers)
	close(release)
	wg.Wait()

	assert.True(t, allWaiting, "all callers must join the in-flight construction")
	assert.Equal(t, int32(callers), constructions.Load(), "every caller must construct an unshareable processor")
	seen := make(map[beat.Processor]struct{}, callers)
	for range callers {
		result := <-results
		assert.NoError(t, result.err, "unshareable construction must succeed")
		_, exists := seen[result.processor]
		assert.False(t, exists, "each caller must receive a distinct processor")
		seen[result.processor] = struct{}{}
	}
}

func TestSafeWrapConcurrentConstructionError(t *testing.T) {
	const (
		name    = "test-concurrent-construction-error"
		callers = 8
	)

	expectedErr := errors.New("construction failed")
	started := make(chan struct{})
	release := make(chan struct{})
	var constructions atomic.Int32
	cons := func(*config.C, *logp.Logger) (beat.Processor, error) {
		if constructions.Add(1) == 1 {
			close(started)
		}
		<-release
		return nil, expectedErr
	}
	sw := SafeWrap(name, cons)
	logger := logptest.NewTestingLogger(t, "")

	results := make(chan error, callers)
	var wg sync.WaitGroup
	wg.Go(func() {
		_, err := sw(nil, logger)
		results <- err
	})
	<-started
	for range callers - 1 {
		wg.Go(func() {
			_, err := sw(nil, logger)
			results <- err
		})
	}

	allWaiting := waitForSharedReferences(t, name, callers)
	close(release)
	wg.Wait()

	assert.True(t, allWaiting, "all callers must join the in-flight construction")
	assert.Equal(t, int32(1), constructions.Load(), "one failed construction must serve all waiting callers")
	for range callers {
		assert.ErrorIs(t, <-results, expectedErr, "all waiting callers must receive the construction error")
	}
}

func TestSafeWrapConcurrentConstructionPanic(t *testing.T) {
	const (
		name    = "test-concurrent-construction-panic"
		callers = 8
	)

	started := make(chan struct{})
	release := make(chan struct{})
	var constructions atomic.Int32
	cons := func(*config.C, *logp.Logger) (beat.Processor, error) {
		if constructions.Add(1) == 1 {
			close(started)
		}
		<-release
		panic("construction panic")
	}
	sw := SafeWrap(name, cons)
	logger := logptest.NewTestingLogger(t, "")

	type result struct {
		err        error
		panicValue any
	}
	results := make(chan result, callers)
	call := func() {
		var err error
		defer func() {
			results <- result{err: err, panicValue: recover()}
		}()
		_, err = sw(nil, logger)
	}

	var wg sync.WaitGroup
	wg.Go(call)
	<-started
	for range callers - 1 {
		wg.Go(call)
	}

	allWaiting := waitForSharedReferences(t, name, callers)
	close(release)
	wg.Wait()

	assert.True(t, allWaiting, "all callers must join the in-flight construction")
	assert.Equal(t, int32(1), constructions.Load(), "waiters must not retry a panicking construction")
	for range callers {
		result := <-results
		assert.Nil(t, result.panicValue, "constructor panics must be converted to errors")
		assert.ErrorContains(t, result.err, "processor constructor panicked: construction panic",
			"all callers must receive the recovered constructor panic")
	}
}

func waitForSharedReferences(t *testing.T, name string, want int) bool {
	t.Helper()
	return assert.Eventually(t, func() bool {
		sharedProcessorMu.Lock()
		defer sharedProcessorMu.Unlock()
		core := sharedProcessors[sharedProcessorKey{name: name}]
		return core != nil && core.refCount == want
	}, time.Second, time.Millisecond, "shared core must reserve every waiting caller's reference")
}

func TestSafeWrapNewInstanceAfterAllRefsClosed(t *testing.T) {
	sw := SafeWrap("test-recreate-after-close", mockCloserConstructor)

	proc1, err := sw(nil, nil)
	require.NoError(t, err, "initial SafeWrap call should succeed")

	require.NoError(t, Close(proc1), "closing the only reference should succeed")

	proc2, err := sw(nil, nil)
	require.NoError(t, err, "SafeWrap after full close should succeed")

	assert.NotSame(t, proc1, proc2, "a new instance must be created after all refs are closed")

	_, err = proc2.Run(nil)
	assert.NoError(t, err, "newly created processor must be runnable")

	require.NoError(t, Close(proc2))
}

func TestSafeWrapUnshareableNotShared(t *testing.T) {
	constructions := 0
	cons := func(cfg *config.C, log *logp.Logger) (beat.Processor, error) {
		constructions++
		return &mockUnshareableProcessor{}, nil
	}
	sw := SafeWrap("test-unshareable", cons)

	proc1, err := sw(nil, nil)
	require.NoError(t, err, "first SafeWrap call should succeed")
	proc2, err := sw(nil, nil)
	require.NoError(t, err, "second SafeWrap call should succeed")

	assert.Equal(t, 2, constructions, "an Unshareable processor must be constructed once per owner")
	assert.NotSame(t, proc1, proc2, "an Unshareable processor must not be shared across owners")
	assert.NotImplements(t, (*Closer)(nil), proc1)
}

func TestSafeWrapForwardsRunPdata(t *testing.T) {
	p := &mockPdataProcessor{}
	cons := func(cfg *config.C, log *logp.Logger) (beat.Processor, error) { return p, nil }
	sw := SafeWrap("test-pdata", cons)

	wp, err := sw(nil, nil)
	require.NoError(t, err, "SafeWrap of a pdata processor should succeed")

	assert.IsType(t, &sharedPdataHandle{}, wp, "a shared PdataProcessor must return a pdata-aware handle")
	pp, ok := wp.(PdataProcessor)
	require.True(t, ok, "shared handle must implement PdataProcessor")

	drop, err := pp.RunPdata(pcommon.NewMap())
	require.NoError(t, err, "RunPdata should forward without error")
	assert.False(t, drop, "mock does not drop the event")
	assert.Equal(t, 1, p.pdataCount, "RunPdata must reach the underlying processor")

	require.NoError(t, Close(wp))
	_, err = pp.RunPdata(pcommon.NewMap())
	assert.ErrorIs(t, err, ErrClosed, "RunPdata must return ErrClosed after the handle is closed")
	assert.Equal(t, 1, p.pdataCount, "RunPdata must not reach the underlying after close")
}

func TestSafeProcessorSetPaths(t *testing.T) {
	cons, p := newMockPathSetterProcessor()
	var (
		bp  beat.Processor
		sp  PathSetter
		err error
	)
	t.Run("creates a wrapped processor", func(t *testing.T) {
		sw := SafeWrap("", cons)
		bp, err = sw(nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 0, p.setPathsCount)
	})

	t.Run("not a closer", func(t *testing.T) {
		assert.NotImplements(t, (*Closer)(nil), p)
		assert.NoError(t, Close(p))
		assert.NoError(t, Close(p))
	})

	t.Run("does not run before SetPaths is called", func(t *testing.T) {
		assert.Equal(t, 0, p.runCount)
		e, err := bp.Run(nil)
		assert.Nil(t, e)
		assert.ErrorIs(t, err, ErrPathsNotSet)
		assert.Equal(t, 0, p.runCount)
	})

	t.Run("sets paths", func(t *testing.T) {
		assert.Equal(t, 0, p.setPathsCount)
		require.Implements(t, (*PathSetter)(nil), bp)
		var ok bool
		sp, ok = bp.(PathSetter)
		require.True(t, ok)
		require.NotNil(t, sp)
		testPaths := &paths.Path{}
		err = sp.SetPaths(testPaths)
		assert.NoError(t, err)
		assert.Equal(t, 1, p.setPathsCount)

		// set paths again with the SAME pointer (idempotent for global processors)
		err = sp.SetPaths(testPaths)
		assert.NoError(t, err)
		assert.Equal(t, 1, p.setPathsCount)

		// set paths again with a DIFFERENT pointer (should error)
		err = sp.SetPaths(&paths.Path{})
		assert.ErrorIs(t, err, ErrPathsAlreadySet)
		assert.Equal(t, 1, p.setPathsCount)
	})

	t.Run("runs after SetPaths is called", func(t *testing.T) {
		assert.Equal(t, 0, p.runCount)
		e, err := bp.Run(nil)
		assert.NoError(t, err)
		assert.Equal(t, e, mockEvent)
		assert.Equal(t, 1, p.runCount)
	})
}
