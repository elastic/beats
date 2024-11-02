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

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/config"
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
	constructor := func(config *config.C) (beat.Processor, error) {
		return &p, nil
	}
	return constructor, &p
}

func mockConstructor(config *config.C) (beat.Processor, error) {
	return &mockProcessor{}, nil
}

func mockCloserConstructor(config *config.C) (beat.Processor, error) {
	return &mockCloserProcessor{}, nil
}

func TestSafeWrap(t *testing.T) {
	t.Run("does not wrap a non-closer processor", func(t *testing.T) {
		nonCloser := mockConstructor
		wrappedNonCloser := SafeWrap(nonCloser)
		wp, err := wrappedNonCloser(nil)
		require.NoError(t, err)
		require.IsType(t, &mockProcessor{}, wp)
	})

	t.Run("wraps a closer processor", func(t *testing.T) {
		closer := mockCloserConstructor
		wrappedCloser := SafeWrap(closer)
		wcp, err := wrappedCloser(nil)
		require.NoError(t, err)
		require.IsType(t, &SafeProcessor{}, wcp)
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
		sp, err = sw(nil)
		require.NoError(t, err)
	})

	t.Run("propagates Run to a processor", func(t *testing.T) {
		require.Equal(t, 0, p.runCount)

		e, err := sp.Run(nil)
		require.NoError(t, err)
		require.Equal(t, e, mockEvent)
		e, err = sp.Run(nil)
		require.NoError(t, err)
		require.Equal(t, e, mockEvent)

		require.Equal(t, 2, p.runCount)
	})

	t.Run("propagates Close to a processor only once", func(t *testing.T) {
		require.Equal(t, 0, p.closeCount)

		err := Close(sp)
		require.NoError(t, err)
		err = Close(sp)
		require.NoError(t, err)

		require.Equal(t, 1, p.closeCount)
	})

	t.Run("does not propagate Run when closed", func(t *testing.T) {
		require.Equal(t, 2, p.runCount) // still 2 from the previous test case
		e, err := sp.Run(nil)
		require.Nil(t, e)
		require.ErrorIs(t, err, ErrClosed)
		require.Equal(t, 2, p.runCount)
	})
}
