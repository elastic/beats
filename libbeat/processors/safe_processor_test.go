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

type mockSetPatherCloserProcessor struct {
	mockCloserProcessor
	setPathsCount int
}

func (p *mockSetPatherCloserProcessor) SetPaths(*paths.Path) error {
	p.setPathsCount++
	return nil
}

func newMockSetPatherCloserProcessor() (Constructor, *mockSetPatherCloserProcessor) {
	p := mockSetPatherCloserProcessor{}
	constructor := func(config *config.C, _ *logp.Logger) (beat.Processor, error) { return &p, nil }
	return constructor, &p
}

type mockSetPatherProcessor struct {
	mockProcessor
	setPathsCount int
}

func (p *mockSetPatherProcessor) SetPaths(*paths.Path) error {
	p.setPathsCount++
	return nil
}

func newMockSetPatherProcessor() (Constructor, *mockSetPatherProcessor) {
	p := mockSetPatherProcessor{}
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
		wrappedNonCloser := SafeWrap(nonCloser)
		wp, err := wrappedNonCloser(nil, nil)
		require.NoError(t, err)
		assert.IsType(t, &mockProcessor{}, wp)
		assert.NotImplements(t, (*Closer)(nil), wp)
	})

	t.Run("wraps a closer processor", func(t *testing.T) {
		closer := mockCloserConstructor
		wrappedCloser := SafeWrap(closer)
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
		sw := SafeWrap(cons)
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
	cons, p := newMockSetPatherCloserProcessor()
	var (
		bp  beat.Processor
		sp  SetPather
		err error
	)
	t.Run("creates a wrapped processor", func(t *testing.T) {
		sw := SafeWrap(cons)
		bp, err = sw(nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 0, p.setPathsCount)
	})

	t.Run("sets paths", func(t *testing.T) {
		assert.Equal(t, 0, p.setPathsCount)
		require.Implements(t, (*SetPather)(nil), bp)
		var ok bool
		sp, ok = bp.(SetPather)
		require.True(t, ok)
		require.NotNil(t, sp)
		err = sp.SetPaths(&paths.Path{})
		assert.NoError(t, err)
		assert.Equal(t, 1, p.setPathsCount)

		// set paths again
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

func TestSafeProcessorSetPaths(t *testing.T) {
	cons, p := newMockSetPatherProcessor()
	var (
		bp  beat.Processor
		sp  SetPather
		err error
	)
	t.Run("creates a wrapped processor", func(t *testing.T) {
		sw := SafeWrap(cons)
		bp, err = sw(nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 0, p.setPathsCount)
	})

	t.Run("not a closer", func(t *testing.T) {
		assert.NotImplements(t, (*Closer)(nil), p)
		assert.NoError(t, Close(p))
		assert.NoError(t, Close(p))
	})

	t.Run("sets paths", func(t *testing.T) {
		assert.Equal(t, 0, p.setPathsCount)
		require.Implements(t, (*SetPather)(nil), bp)
		var ok bool
		sp, ok = bp.(SetPather)
		require.True(t, ok)
		require.NotNil(t, sp)
		err = sp.SetPaths(&paths.Path{})
		assert.NoError(t, err)
		assert.Equal(t, 1, p.setPathsCount)

		// set paths again
		err = sp.SetPaths(&paths.Path{})
		assert.ErrorIs(t, err, ErrPathsAlreadySet)
		assert.Equal(t, 1, p.setPathsCount)
	})
}
