// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin || synthetics

package api

import (
	"maps"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v9/heartbeat/ecserr"
	"github.com/elastic/beats/v9/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v9/libbeat/beat"
	"github.com/elastic/beats/v9/x-pack/heartbeat/monitors/browser"
)

// The api plugin must be discoverable under its canonical name and alias, and
// carry a config hash func so Fleet parameter pushes don't cause stop/restarts.
func TestAPIPluginRegistered(t *testing.T) {
	for _, name := range []string{"api", "synthetics/api"} {
		t.Run(name, func(t *testing.T) {
			factory, found := plugin.GlobalPluginsReg.Get(name)
			require.Truef(t, found, "%q must be registered", name)
			assert.Equal(t, "api", factory.Name, "alias must resolve to the api plugin")
			assert.NotNil(t, factory.HashConfig, "api must register a params-insensitive hash func")
		})
	}
}

func apiConfig(t *testing.T) *conf.C {
	t.Helper()
	cfg, err := conf.NewConfigFrom(mapstr.M{
		"type":     "api",
		"name":     "My API monitor",
		"id":       "myApiId",
		"schedule": "@every 1m",
		"source": mapstr.M{
			"inline": mapstr.M{
				"script": "// api journey",
			},
		},
	})
	require.NoError(t, err)
	return cfg
}

// API journeys run the same Node.js agent as browser monitors, so they must
// honor the ELASTIC_SYNTHETICS_CAPABLE gate rather than skip it.
func TestAPICreateRequiresSyntheticsCapable(t *testing.T) {
	t.Setenv("ELASTIC_SYNTHETICS_CAPABLE", "")

	_, err := create("api", apiConfig(t), beat.Info{})
	require.Error(t, err, "create must fail when ELASTIC_SYNTHETICS_CAPABLE is unset")

	var ecsErr *ecserr.ECSErr
	require.ErrorAs(t, err, &ecsErr, "gate error must be an ECSErr")
	assert.Equal(t, ecserr.ECode("AGENT_NOT_BROWSER_CAPABLE"), ecsErr.Code,
		"gate must return the not-synthetics-capable error")
}

// When the gate is open, create builds the plugin off the shared browser
// pipeline. It must not return the not-capable error.
func TestAPICreatePassesGateWhenCapable(t *testing.T) {
	if syscall.Geteuid() == 0 {
		t.Skip("skipping: create's root guard rejects euid 0, unrelated to the gate")
	}
	t.Setenv("ELASTIC_SYNTHETICS_CAPABLE", "true")

	p, err := create("api", apiConfig(t), beat.Info{})
	require.NoError(t, err, "create must succeed once the gate is open")
	assert.Equal(t, 1, p.Endpoints, "api monitor is a single synthetics endpoint")
}

// The api plugin must hash configs identically to browser (params excluded), so
// pushing new params from Fleet is a no-op reload rather than a restart.
func TestAPIHashIgnoresParams(t *testing.T) {
	base := mapstr.M{
		"type": "api",
		"name": "My API monitor",
		"id":   "myApiId",
		"source": mapstr.M{
			"inline": mapstr.M{"script": "// api journey"},
		},
	}
	withParams := mapstr.M{}
	maps.Copy(withParams, base)
	withParams["params"] = mapstr.M{"key": "value"}

	factory, found := plugin.GlobalPluginsReg.Get("api")
	require.True(t, found)

	h1, err := factory.HashConfig(conf.MustNewConfigFrom(base))
	require.NoError(t, err)
	h2, err := factory.HashConfig(conf.MustNewConfigFrom(withParams))
	require.NoError(t, err)
	assert.Equal(t, h1, h2, "params must not affect the config hash")

	// Sanity check the plugin reuses browser's hash implementation.
	hBrowser, err := browser.HashConfig(conf.MustNewConfigFrom(base))
	require.NoError(t, err)
	assert.Equal(t, hBrowser, h1, "api must reuse browser's params-insensitive hash")
}
