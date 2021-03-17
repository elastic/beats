// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"io/ioutil"
	"testing"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
)

func TestConfig(t *testing.T) {
	testMgmtMode(t)
	testLocalConfig(t)
}

func testMgmtMode(t *testing.T) {
	t.Run("succeed when local mode is selected", func(t *testing.T) {
		c := mustWithConfigMode(true)
		m := localConfig{}
		err := c.Unpack(&m)
		require.NoError(t, err)
		assert.Equal(t, false, m.Fleet.Enabled)
		assert.Equal(t, true, configuration.IsStandalone(m.Fleet))

	})

	t.Run("succeed when fleet mode is selected", func(t *testing.T) {
		c := mustWithConfigMode(false)
		m := localConfig{}
		err := c.Unpack(&m)
		require.NoError(t, err)
		assert.Equal(t, true, m.Fleet.Enabled)
		assert.Equal(t, false, configuration.IsStandalone(m.Fleet))
	})
}

func testLocalConfig(t *testing.T) {
	t.Run("only accept positive period", func(t *testing.T) {
		c := config.MustNewConfigFrom(map[string]interface{}{
			"enabled": true,
			"period":  0,
		})

		m := configuration.ReloadConfig{}
		err := c.Unpack(&m)
		assert.Error(t, err)

		c = config.MustNewConfigFrom(map[string]interface{}{
			"enabled": true,
			"period":  1,
		})

		err = c.Unpack(&m)
		assert.NoError(t, err)
		assert.Equal(t, 1*time.Second, m.Period)
	})
}

func mustWithConfigMode(standalone bool) *config.Config {
	return config.MustNewConfigFrom(
		map[string]interface{}{
			"fleet": map[string]interface{}{
				"enabled":        !standalone,
				"kibana":         map[string]interface{}{"host": "demo"},
				"access_api_key": "123",
			},
		},
	)
}

func dumpToYAML(t *testing.T, out string, in interface{}) {
	b, err := yaml.Marshal(in)
	require.NoError(t, err)
	ioutil.WriteFile(out, b, 0600)
}
