// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type mockHandler struct {
	received action
	called   bool
	err      error
}

func (h *mockHandler) Handle(a action, acker fleetAcker) error {
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
	ack := newNoopAcker()

	t.Run("Success to dispatch multiples events", func(t *testing.T) {
		def := &mockHandler{}
		d, err := newActionDispatcher(nil, def)
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

	t.Run("Unknown action are catched by the unknown handler", func(t *testing.T) {
		def := &mockHandler{}
		d, err := newActionDispatcher(nil, def)
		require.NoError(t, err)

		action := &mockActionUnknown{}
		err = d.Dispatch(ack, action)

		require.True(t, def.called)
		require.Equal(t, action, def.received)
	})

	t.Run("Could not register two handlers on the same action", func(t *testing.T) {
		success1 := &mockHandler{}
		success2 := &mockHandler{}

		def := &mockHandler{}
		d, err := newActionDispatcher(nil, def)
		require.NoError(t, err)

		err = d.Register(&mockAction{}, success1)
		require.NoError(t, err)

		err = d.Register(&mockAction{}, success2)
		require.Error(t, err)
	})
}
