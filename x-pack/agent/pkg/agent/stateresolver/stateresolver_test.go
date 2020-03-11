// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stateresolver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/logger"
)

func TestStateResolverAcking(t *testing.T) {
	submit := &cfg{
		id:        "config-1",
		createdAt: time.Now(),
		programs: []program.Program{
			fb("1"), mb("1"),
		},
	}

	t.Run("when we ACK the should state", func(t *testing.T) {
		log, _ := logger.New()
		r, err := NewStateResolver(log)
		require.NoError(t, err)

		// Current state is empty.
		_, steps, ack, err := r.Resolve(submit)
		require.NoError(t, err)
		require.Equal(t, 2, len(steps))

		// Ack the should state.
		ack()

		// Current sate is not empty lets try to resolve the same configuration.
		_, steps, ack, err = r.Resolve(submit)
		require.NoError(t, err)
		require.Equal(t, 0, len(steps))
	})

	t.Run("when we don't ACK the should state", func(t *testing.T) {
		log, _ := logger.New()
		r, err := NewStateResolver(log)
		require.NoError(t, err)

		// Current state is empty.
		_, steps1, _, err := r.Resolve(submit)
		require.NoError(t, err)
		require.Equal(t, 2, len(steps1))

		// We didn't ACK the should state, verify that resolve produce the same output.
		_, steps2, _, err := r.Resolve(submit)
		require.NoError(t, err)
		require.Equal(t, 2, len(steps2))

		assert.Equal(t, steps1, steps2)
	})
}
