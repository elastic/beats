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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
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

func mockConstructor(config *config.C, log *logp.Logger) (beat.Processor, error) {
	return &mockProcessor{}, nil
}

func mockCloserConstructor(config *config.C, log *logp.Logger) (beat.Processor, error) {
	return &mockCloserProcessor{}, nil
}

func TestSafeWrap(t *testing.T) {
	t.Run("does not wrap a non-closer processor", func(t *testing.T) {
		nonCloser := mockConstructor
		wrappedNonCloser := SafeWrap("non-closer processor", nonCloser)
		wp, err := wrappedNonCloser(nil, nil)
		require.NoError(t, err)
		assert.IsType(t, &mockProcessor{}, wp)
		assert.NotImplements(t, (*Closer)(nil), wp)
	})

	t.Run("wraps a closer processor", func(t *testing.T) {
		closer := mockCloserConstructor
		wrappedCloser := SafeWrap("closer processor", closer)
		wcp, err := wrappedCloser(nil, nil)
		require.NoError(t, err)
		assert.IsType(t, &safeProcessorWithClose{}, wcp)
		assert.Implements(t, (*Closer)(nil), wcp)
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
	sw := SafeWrap("test-shared-instance", cons)

	proc1, err := sw(nil, nil)
	require.NoError(t, err, "first SafeWrap call should succeed")

	proc2, err := sw(nil, nil)
	require.NoError(t, err, "second SafeWrap call should succeed")

	assert.Same(t, proc1, proc2, "same name+config should return the same processor pointer")

	_, err = proc1.Run(nil)
	require.NoError(t, err, "Run via proc1 should succeed")
	assert.Equal(t, 1, p.runCount, "run should be reflected in the underlying mock")

	require.NoError(t, Close(proc1), "first Close should not error")
	assert.Equal(t, 0, p.closeCount, "underlying processor should not be closed while a ref remains")

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

func TestSafeWrapNewInstanceAfterAllRefsClosed(t *testing.T) {
	sw := SafeWrap("test-recreate-after-close", mockCloserConstructor)

	proc1, err := sw(nil, nil)
	require.NoError(t, err, "initial SafeWrap call should succeed")

	require.NoError(t, Close(proc1), "closing the only reference should succeed")

	// Entry is removed from sharedProcessors; next call must build a fresh instance.
	proc2, err := sw(nil, nil)
	require.NoError(t, err, "SafeWrap after full close should succeed")

	assert.NotSame(t, proc1, proc2, "a new instance must be created after all refs are closed")

	_, err = proc2.Run(nil)
	assert.NoError(t, err, "newly created processor must be runnable")

	require.NoError(t, Close(proc2))
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
