// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
)

func TestLoadConfig(t *testing.T) {
	contents := map[string]interface{}{
		"outputs": map[string]interface{}{
			"default": map[string]interface{}{
				"type":     "elasticsearch",
				"hosts":    []interface{}{"127.0.0.1:9200"},
				"username": "elastic",
				"password": "changeme",
			},
		},
		"inputs": []interface{}{
			map[string]interface{}{
				"type": "logfile",
				"streams": []interface{}{
					map[string]interface{}{
						"paths": []interface{}{"/var/log/${host.name}"},
					},
				},
			},
		},
	}

	tmp, err := ioutil.TempDir("", "config")
	require.NoError(t, err)
	defer os.RemoveAll(tmp)

	cfgPath := filepath.Join(tmp, "config.yml")
	dumpToYAML(t, cfgPath, contents)

	cfg, err := LoadConfigFromFile(cfgPath)
	require.NoError(t, err)

	cfgData, err := cfg.ToMapStr()
	require.NoError(t, err)

	assert.Equal(t, contents, cfgData)
}

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
		assert.Equal(t, true, isStandalone(m.Fleet))

	})

	t.Run("succeed when fleet mode is selected", func(t *testing.T) {
		c := mustWithConfigMode(false)
		m := localConfig{}
		err := c.Unpack(&m)
		require.NoError(t, err)
		assert.Equal(t, true, m.Fleet.Enabled)
		assert.Equal(t, false, isStandalone(m.Fleet))
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
