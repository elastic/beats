// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package dispatcher

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage/store"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
	noopacker "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi/acker/noop"
)

type mockHandler struct {
	received fleetapi.Action
	called   bool
	err      error
}

func (h *mockHandler) Handle(_ context.Context, a fleetapi.Action, acker store.FleetAcker) error {
	h.called = true
	h.received = a
	return h.err
}

type mockAction struct{}

func (m *mockAction) ID() string     { return "mockAction" }
func (m *mockAction) Type() string   { return "mockAction" }
func (m *mockAction) String() string { return "mockAction" }

type mockActionUnknown struct{}

func (m *mockActionUnknown) ID() string     { return "mockActionUnknown" }
func (m *mockActionUnknown) Type() string   { return "mockActionUnknown" }
func (m *mockActionUnknown) String() string { return "mockActionUnknown" }

type mockActionOther struct{}

func (m *mockActionOther) ID() string     { return "mockActionOther" }
func (m *mockActionOther) Type() string   { return "mockActionOther" }
func (m *mockActionOther) String() string { return "mockActionOther" }

func TestActionDispatcher(t *testing.T) {
	ack := noopacker.NewAcker()

	t.Run("Success to dispatch multiples events", func(t *testing.T) {
		def := &mockHandler{}
		d, err := New(context.Background(), nil, def)
		require.NoError(t, err)

		success1 := &mockHandler{}
		success2 := &mockHandler{}

		d.Register(&mockAction{}, success1)
		d.Register(&mockActionOther{}, success2)

		action1 := &mockAction{}
		action2 := &mockActionOther{}

		err = d.Dispatch(ack, action1, action2)

		require.NoError(t, err)

		require.True(t, success1.called)
		require.Equal(t, action1, success1.received)

		require.True(t, success2.called)
		require.Equal(t, action2, success2.received)

		require.False(t, def.called)
		require.Nil(t, def.received)
	})

	t.Run("Unknown action are caught by the unknown handler", func(t *testing.T) {
		def := &mockHandler{}
		d, err := New(context.Background(), nil, def)
		require.NoError(t, err)

		action := &mockActionUnknown{}
		err = d.Dispatch(ack, action)

		require.NoError(t, err)
		require.True(t, def.called)
		require.Equal(t, action, def.received)
	})

	t.Run("Could not register two handlers on the same action", func(t *testing.T) {
		success1 := &mockHandler{}
		success2 := &mockHandler{}

		def := &mockHandler{}
		d, err := New(context.Background(), nil, def)
		require.NoError(t, err)

		err = d.Register(&mockAction{}, success1)
		require.NoError(t, err)

		err = d.Register(&mockAction{}, success2)
		require.Error(t, err)
	})
}
