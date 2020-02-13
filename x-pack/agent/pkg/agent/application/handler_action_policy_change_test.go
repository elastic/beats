// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
	"sync"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/x-pack/agent/pkg/config"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/fleetapi"
)

type mockEmitter struct {
	err    error
	policy *config.Config
}

func (m *mockEmitter) Emitter(policy *config.Config) error {
	m.policy = policy
	return m.err
}

func TestPolicyChange(t *testing.T) {
	log, _ := logger.New()
	ack := newNoopAcker()

	t.Run("Receive a policy change and successfully emits a raw configuration", func(t *testing.T) {
		emitter := &mockEmitter{}

		policy := map[string]interface{}{"hello": "world"}
		action := &fleetapi.ActionPolicyChange{
			ActionID:   "abc123",
			ActionType: "POLICY_CHANGE",
			Policy:     policy,
		}

		handler := &handlerPolicyChange{log: log, emitter: emitter.Emitter}

		err := handler.Handle(context.Background(), action, ack)
		require.NoError(t, err)
		require.Equal(t, config.MustNewConfigFrom(policy), emitter.policy)
	})

	t.Run("Receive a policy and fail to emits a raw configuration", func(t *testing.T) {
		mockErr := errors.New("error returned")
		emitter := &mockEmitter{err: mockErr}

		policy := map[string]interface{}{"hello": "world"}
		action := &fleetapi.ActionPolicyChange{
			ActionID:   "abc123",
			ActionType: "POLICY_CHANGE",
			Policy:     policy,
		}

		handler := &handlerPolicyChange{log: log, emitter: emitter.Emitter}

		err := handler.Handle(context.Background(), action, ack)
		require.Error(t, err)
	})
}

func TestPolicyAcked(t *testing.T) {
	log, _ := logger.New()
	t.Run("Policy change should not ACK on error", func(t *testing.T) {
		tacker := &testAcker{}

		mockErr := errors.New("error returned")
		emitter := &mockEmitter{err: mockErr}

		policy := map[string]interface{}{"hello": "world"}
		actionID := "abc123"
		action := &fleetapi.ActionPolicyChange{
			ActionID:   actionID,
			ActionType: "POLICY_CHANGE",
			Policy:     policy,
		}

		handler := &handlerPolicyChange{log: log, emitter: emitter.Emitter}

		err := handler.Handle(context.Background(), action, tacker)
		require.Error(t, err)

		actions := tacker.Items()
		assert.EqualValues(t, 0, len(actions))
	})

	t.Run("Policy change should ACK", func(t *testing.T) {
		tacker := &testAcker{}

		emitter := &mockEmitter{}

		policy := map[string]interface{}{"hello": "world"}
		actionID := "abc123"
		action := &fleetapi.ActionPolicyChange{
			ActionID:   actionID,
			ActionType: "POLICY_CHANGE",
			Policy:     policy,
		}

		handler := &handlerPolicyChange{log: log, emitter: emitter.Emitter}

		err := handler.Handle(context.Background(), action, tacker)
		require.NoError(t, err)

		actions := tacker.Items()
		assert.EqualValues(t, 1, len(actions))
		assert.Equal(t, actionID, actions[0])
	})
}

type testAcker struct {
	acked     []string
	ackedLock sync.Mutex
}

func (t *testAcker) Ack(_ context.Context, action fleetapi.Action) error {
	t.ackedLock.Lock()
	defer t.ackedLock.Unlock()

	if t.acked == nil {
		t.acked = make([]string, 0)
	}

	t.acked = append(t.acked, action.ID())
	return nil
}

func (t *testAcker) Commit(_ context.Context) error {
	return nil
}

func (t *testAcker) Clear() {
	t.ackedLock.Lock()
	defer t.ackedLock.Unlock()

	t.acked = make([]string, 0)
}

func (t *testAcker) Items() []string {
	t.ackedLock.Lock()
	defer t.ackedLock.Unlock()
	return t.acked
}
