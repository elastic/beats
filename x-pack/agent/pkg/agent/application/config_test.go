// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/config"
)

func TestConfig(t *testing.T) {
	testMgmtMode(t)
	testLocalConfig(t)
}

func testMgmtMode(t *testing.T) {
	t.Run("succeed when local mode is selected", func(t *testing.T) {
		c := mustWithConfigMode("local")
		m := ManagementConfig{}
		err := c.Unpack(&m)
		require.NoError(t, err)
		assert.Equal(t, localMode, m.Mode)

	})

	t.Run("succeed when fleet mode is selected", func(t *testing.T) {
		c := mustWithConfigMode("fleet")
		m := ManagementConfig{}
		err := c.Unpack(&m)
		require.NoError(t, err)
		assert.Equal(t, fleetMode, m.Mode)
	})

	t.Run("fails on unknown mode", func(t *testing.T) {
		c := mustWithConfigMode("what")
		m := ManagementConfig{}
		err := c.Unpack(&m)
		require.Error(t, err)
	})
}

func testLocalConfig(t *testing.T) {
	t.Run("only accept positive period", func(t *testing.T) {
		c := config.MustNewConfigFrom(map[string]interface{}{
			"enabled": true,
			"period":  0,
		})

		m := reloadConfig{}
		err := c.Unpack(&m)
		require.Error(t, err)

		c = config.MustNewConfigFrom(map[string]interface{}{
			"enabled": true,
			"period":  1,
		})

		err = c.Unpack(&m)
		require.NoError(t, err)
	})
}

func mustWithConfigMode(m string) *config.Config {
	return config.MustNewConfigFrom(
		map[string]interface{}{
			"mode": m,
		},
	)
}
