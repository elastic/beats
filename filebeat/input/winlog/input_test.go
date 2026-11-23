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

package winlog

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/monitoring"
	"github.com/elastic/beats/v7/winlogbeat/checkpoint"
	"github.com/elastic/beats/v7/winlogbeat/eventlog"
)

var errBoom = errors.New("boom")

// stubLog is an eventlog.EventLog whose Read always returns readErr.
type stubLog struct {
	readErr error
}

func (s stubLog) Open(checkpoint.EventLogState, *monitoring.Registry) error { return nil }
func (s stubLog) Checkpoint() checkpoint.EventLogState                      { return checkpoint.EventLogState{} }
func (s stubLog) Read() ([]eventlog.Record, error)                          { return nil, s.readErr }
func (s stubLog) Reset() error                                              { return nil }
func (s stubLog) Close() error                                              { return nil }
func (s stubLog) Name() string                                              { return "Security" }
func (s stubLog) Channel() string                                           { return "Security" }
func (s stubLog) IsFile() bool                                              { return false }
func (s stubLog) IgnoreMissingChannel() bool                                { return false }

type nopPublisher struct{}

func (nopPublisher) Publish([]eventlog.Record) error { return nil }

func TestStartState(t *testing.T) {
	log := logp.NewNopLogger()
	legacy := checkpoint.EventLogState{Name: "Security", RecordNumber: 42, Bookmark: "<b/>"}
	stored := checkpoint.EventLogState{Name: "Security", RecordNumber: 99}
	withLegacy := winlogInput{opts: Options{InitialState: func(n string) (checkpoint.EventLogState, bool) {
		return legacy, n == "Security"
	}}}
	unpackStored := func(to any) error { *(to.(*checkpoint.EventLogState)) = stored; return nil }
	unpackFail := func(any) error { return errors.New("bad") }

	assert.Equal(t, legacy, withLegacy.startState(log, "Security", true, unpackStored),
		"no cursor and a legacy entry should start from the legacy state")
	assert.Equal(t, stored, withLegacy.startState(log, "Security", false, unpackStored),
		"the cursor should win over the legacy state")
	assert.Equal(t, checkpoint.EventLogState{}, withLegacy.startState(log, "Application", true, unpackStored),
		"no cursor and no legacy entry for this name should start from zero")
	assert.Equal(t, checkpoint.EventLogState{}, winlogInput{}.startState(log, "Security", true, unpackStored),
		"no hook should start from zero")
	assert.Equal(t, checkpoint.EventLogState{}, withLegacy.startState(log, "Security", false, unpackFail),
		"an unreadable cursor should reset to zero without using the legacy state")
}

func TestRunOnStopped(t *testing.T) {
	newCtx := func(ctx context.Context) input.Context {
		return input.Context{
			Logger:          logp.NewNopLogger(),
			Cancelation:     ctx,
			MetricsRegistry: monitoring.NewRegistry(),
		}
	}
	hooked := func(calls *[]string) winlogInput {
		return winlogInput{opts: Options{OnStopped: func(n string) { *calls = append(*calls, n) }}}
	}
	start := checkpoint.EventLogState{}

	t.Run("EOF calls hook", func(t *testing.T) {
		var calls []string
		err := hooked(&calls).run(newCtx(context.Background()), stubLog{readErr: io.EOF}, start, nopPublisher{})
		assert.NoError(t, err, "EOF should end the run cleanly")
		assert.Equal(t, []string{"Security"}, calls, "hook should be called once")
	})

	t.Run("read error calls hook", func(t *testing.T) {
		var calls []string
		err := hooked(&calls).run(newCtx(context.Background()), stubLog{readErr: errBoom}, start, nopPublisher{})
		assert.ErrorIs(t, err, errBoom, "run should return the read error")
		assert.Equal(t, []string{"Security"}, calls, "hook should be called once")
	})

	t.Run("cancelled does not call hook", func(t *testing.T) {
		var calls []string
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := hooked(&calls).run(newCtx(ctx), stubLog{readErr: io.EOF}, start, nopPublisher{})
		assert.NoError(t, err, "a cancelled run should end cleanly")
		assert.Empty(t, calls, "hook should not be called after cancellation")
	})

	t.Run("no hook", func(t *testing.T) {
		err := winlogInput{}.run(newCtx(context.Background()), stubLog{readErr: io.EOF}, start, nopPublisher{})
		assert.NoError(t, err, "EOF without a hook should end the run cleanly")
	})
}
