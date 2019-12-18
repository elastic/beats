// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"testing"

	"github.com/pkg/errors"
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
	t.Run("Receive a policy change and successfully emits a raw configuration", func(t *testing.T) {
		emitter := &mockEmitter{}

		policy := map[string]interface{}{"hello": "world"}
		action := &fleetapi.ActionPolicyChange{
			ActionBase: &fleetapi.ActionBase{ActionID: "abc123", ActionType: "POLICY_CHANGE"},
			Policy:     policy,
		}

		handler := &handlerPolicyChange{log: log, emitter: emitter.Emitter}

		err := handler.Handle(action)
		require.NoError(t, err)
		require.Equal(t, config.MustNewConfigFrom(policy), emitter.policy)
	})

	t.Run("Receive a policy and fail to emits a raw configuration", func(t *testing.T) {
		mockErr := errors.New("error returned")
		emitter := &mockEmitter{err: mockErr}

		policy := map[string]interface{}{"hello": "world"}
		action := &fleetapi.ActionPolicyChange{
			ActionBase: &fleetapi.ActionBase{ActionID: "abc123", ActionType: "POLICY_CHANGE"},
			Policy:     policy,
		}

		handler := &handlerPolicyChange{log: log, emitter: emitter.Emitter}

		err := handler.Handle(action)
		require.Error(t, err)
	})
}
