// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kvstore

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/unison"
)

func configureOkay() func(cfg *config.C) (Input, error) {
	return func(cfg *config.C) (Input, error) {
		return &testInput{}, nil
	}
}

func configureErr() func(cfg *config.C) (Input, error) {
	return func(cfg *config.C) (Input, error) {
		return nil, errors.New("test error")
	}
}

func TestManager_Create(t *testing.T) {
	t.Run("create-ok", func(t *testing.T) {
		m := Manager{
			Logger:    logp.L(),
			Type:      "test",
			Configure: configureOkay(),
		}

		c, err := config.NewConfigFrom(&managerConfig{ID: "create-ok"})
		require.NoError(t, err)

		_, gotErr := m.Create(c)
		require.NoError(t, gotErr)
	})

	t.Run("err-configure", func(t *testing.T) {
		m := Manager{
			Logger:    logp.L(),
			Type:      "test",
			Configure: configureErr(),
		}

		c, err := config.NewConfigFrom(&managerConfig{ID: "err-configure"})
		require.NoError(t, err)

		_, gotErr := m.Create(c)
		require.ErrorContains(t, gotErr, "test error")
	})

	t.Run("err-config-unpack", func(t *testing.T) {
		m := Manager{
			Logger:    logp.L(),
			Type:      "test",
			Configure: configureOkay(),
		}

		emptyCfg := struct{}{}
		c, err := config.NewConfigFrom(&emptyCfg)
		require.NoError(t, err)

		_, gotErr := m.Create(c)
		require.ErrorContains(t, gotErr, "string value is not set accessing 'id'")
	})
}

func TestManager_Init(t *testing.T) {
	var grp unison.TaskGroup

	m := Manager{}
	gotErr := m.Init(&grp)

	require.NoError(t, gotErr)
}
