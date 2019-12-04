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

func (h *mockHandler) Handle(a action) error {
	h.called = true
	h.received = a
	return h.err
}

type mockAction struct{}
type mockActionUnknown struct{}
type mockActionOther struct{}

func TestActionDispatcher(t *testing.T) {
	t.Run("Success to dispatch multiples events", func(t *testing.T) {
		def := &mockHandler{}
		dispatcher, err := newActionDispatcher(nil, def)
		require.NoError(t, err)

		success1 := &mockHandler{}
		success2 := &mockHandler{}

		dispatcher.Register(&mockAction{}, success1)
		dispatcher.Register(&mockActionOther{}, success2)

		action1 := &mockAction{}
		action2 := &mockActionOther{}

		err = dispatcher.Dispatch(action1, action2)

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
		dispatcher, err := newActionDispatcher(nil, def)
		require.NoError(t, err)

		success := &mockHandler{}
		dispatcher.Register(mockAction{}, success)

		action := &mockActionUnknown{}
		err = dispatcher.Dispatch(action)

		require.NoError(t, err)
		require.False(t, success.called)

		require.True(t, def.called)
		require.Equal(t, action, def.received)

		require.False(t, success.called)
		require.Nil(t, success.received)
	})
}
