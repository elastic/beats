// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package beater

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
)

// noQueriesConfig is the config for a policy with no inputs.
const noQueriesConfig = `{
    "options": {
        "schedule_max_drift": 60,
        "schedule_splay_percent": 10
    }
}`

// TestConfigPluginNoFIPSLeavesOptionsAlone checks that builds without the
// requirefips tag pass the policy's options to osqueryd as they are.
func TestConfigPluginNoFIPSLeavesOptionsAlone(t *testing.T) {
	cp := NewConfigPlugin(logptest.NewTestingLogger(t, "config_test"))
	err := cp.Set([]config.InputConfig{{
		Name: "osquery-manager-1",
		Type: "osquery",
		Osquery: &config.OsqueryConfig{Options: map[string]any{
			"disable_distributed":      false,
			"yara_sigurl_authenticate": true,
		}},
	}})
	require.NoError(t, err, "setting the policy")

	generated, err := cp.GenerateConfig(context.Background())
	require.NoError(t, err, "generating the osquery config")

	var cfg struct {
		Options map[string]any `json:"options"`
	}
	err = json.Unmarshal([]byte(generated[configName]), &cfg)
	require.NoError(t, err, "decoding the osquery config")

	want := map[string]any{
		"disable_distributed":      false,
		"yara_sigurl_authenticate": true,
		"schedule_splay_percent":   float64(10),
		"schedule_max_drift":       float64(60),
	}
	assert.Equal(t, want, cfg.Options, "options must pass through in builds without requirefips")
}
