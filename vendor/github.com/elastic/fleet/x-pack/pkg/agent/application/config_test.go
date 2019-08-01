// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/fleet/x-pack/pkg/config"
)

func TestConfig(t *testing.T) {
	t.Run("Unpack delegates unknown configuration", func(t *testing.T) {
		c := map[string]interface{}{
			"inputs": []map[string]interface{}{
				map[string]interface{}{
					"type":  "log/file",
					"paths": []string{"/var/log/hello.log", "/var/log/bye.log"},
				},
				map[string]interface{}{
					"type":  "log/journal",
					"paths": []string{"/var/log/hello.log", "/var/log/bye.log"},
				},
			},
		}

		cfg := config.MustNewConfigFrom(c)

		config := Config{}
		err := cfg.Unpack(&config)
		require.NoError(t, err)
		assert.Equal(t, 2, len(config.Inputs))

		type subcfg struct {
			Paths []string `config:"paths"`
		}

		c1 := &subcfg{}
		err = config.Inputs[0].RawDelegateConfig.Unpack(&c1)
		require.NoError(t, err)
		assert.Equal(t, 2, len(c1.Paths))

		c2 := &subcfg{}
		err = config.Inputs[1].RawDelegateConfig.Unpack(&c2)
		require.NoError(t, err)
		assert.Equal(t, 2, len(c2.Paths))
	})

	t.Run("Inputs must have a type", func(t *testing.T) {
		c := map[string]interface{}{
			"inputs": []map[string]interface{}{
				map[string]interface{}{
					"paths": []string{"/var/log/hello.log", "/var/log/bye.log"},
				},
			},
		}

		cfg := config.MustNewConfigFrom(c)

		config := Config{}
		err := cfg.Unpack(&config)
		require.Error(t, err)
	})
}
