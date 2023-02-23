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
	runCount   int
	closeCount int
}

func newMockConstructor() (Constructor, *mockProcessor) {
	p := mockProcessor{}
	constructor := func(config *config.C) (Processor, error) {
		return &p, nil
	}
	return constructor, &p
}

func (p *mockProcessor) Run(event *beat.Event) (*beat.Event, error) {
	p.runCount++
	return mockEvent, nil
}

func (p *mockProcessor) Close() error {
	p.closeCount++
	return nil
}
func (p *mockProcessor) String() string {
	return "mock-processor"
}

func TestSafeProcessor(t *testing.T) {
	cons, p := newMockConstructor()
	var (
		sp  Processor
		err error
	)
	t.Run("creates a wrapped processor", func(t *testing.T) {
		sw := SafeWrap(cons)
		sp, err = sw(nil)
		require.NoError(t, err)
	})

	t.Run("propagates Run to a processor", func(t *testing.T) {
		e, err := sp.Run(nil)
		require.NoError(t, err)
		require.Equal(t, e, mockEvent)

		e, err = sp.Run(nil)
		require.NoError(t, err)
		require.Equal(t, e, mockEvent)

		require.Equal(t, 2, p.runCount)
	})

	t.Run("propagates Close to a processor only once", func(t *testing.T) {
		err := Close(sp)
		require.NoError(t, err)

		err = Close(sp)
		require.NoError(t, err)

		require.Equal(t, 1, p.closeCount)
	})

	t.Run("does not propagate Run when closed", func(t *testing.T) {
		e, err := sp.Run(nil)
		require.Nil(t, e)
		require.ErrorIs(t, err, ErrClosed)
		require.Equal(t, 2, p.runCount) // still 2 from the previous test case
	})
}
