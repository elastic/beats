// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build requirefips

package beater

import (
	"context"
	"encoding/json"
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
)

// wantFIPSOptions is spelled out here, not taken from the osqd package, so that
// removing an entry there fails this test.
var wantFIPSOptions = map[string]any{
	"disable_distributed":      true,
	"disable_enrollment":       true,
	"disable_carver":           true,
	"aws_enforce_fips":         true,
	"yara_sigurl_authenticate": false,
}

// noQueriesConfig is the config for a policy with no inputs: the native option
// defaults and the options that FIPS builds enforce.
const noQueriesConfig = `{
    "options": {
        "aws_enforce_fips": true,
        "disable_carver": true,
        "disable_distributed": true,
        "disable_enrollment": true,
        "schedule_max_drift": 60,
        "schedule_splay_percent": 10,
        "yara_sigurl_authenticate": false
    }
}`

func TestConfigPluginFIPSOptions(t *testing.T) {
	tests := []struct {
		name    string
		options map[string]any
		// others are options expected in the config besides the FIPS options.
		// Numbers are float64 because the test reads the config back from JSON.
		others map[string]any
	}{
		{
			name: "no_options",
			others: map[string]any{
				"schedule_splay_percent": float64(10),
				"schedule_max_drift":     float64(60),
			},
		},
		{
			name: "policy_turns_remote_features_on",
			options: map[string]any{
				"disable_distributed":      false,
				"disable_enrollment":       false,
				"disable_carver":           false,
				"aws_enforce_fips":         false,
				"yara_sigurl_authenticate": true,
			},
			others: map[string]any{
				"schedule_splay_percent": float64(10),
				"schedule_max_drift":     float64(60),
			},
		},
		{
			name: "other_options_pass_through",
			options: map[string]any{
				"disable_distributed":    false,
				"schedule_splay_percent": 5,
			},
			others: map[string]any{
				"schedule_splay_percent": float64(5),
				"schedule_max_drift":     float64(60),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := generatedOptions(t, tc.options)

			want := make(map[string]any)
			maps.Copy(want, wantFIPSOptions)
			maps.Copy(want, tc.others)
			assert.Equal(t, want, got, "options in the osquery config for policy options %v", tc.options)
		})
	}
}

// TestConfigPluginFIPSLeavesPolicyOptionsAlone checks that rendering the config
// does not change the policy's own options. The runner reads the same map to
// build the osqueryd flags and to tell when the flags change.
func TestConfigPluginFIPSLeavesPolicyOptionsAlone(t *testing.T) {
	options := map[string]any{"disable_distributed": false}

	generatedOptions(t, options)

	assert.Equal(t, false, options["disable_distributed"], "the policy's disable_distributed must not change")
	assert.NotContains(t, options, "disable_enrollment", "enforced options must not be added to the policy's options")
}

// generatedOptions returns the options section of the osquery config that a
// config plugin generates for a policy with the given options.
func generatedOptions(t *testing.T, options map[string]any) map[string]any {
	t.Helper()

	cp := NewConfigPlugin(logptest.NewTestingLogger(t, "config_test"))
	err := cp.Set([]config.InputConfig{{
		Name:    "osquery-manager-1",
		Type:    "osquery",
		Osquery: &config.OsqueryConfig{Options: options},
	}})
	require.NoError(t, err, "setting the policy")

	generated, err := cp.GenerateConfig(context.Background())
	require.NoError(t, err, "generating the osquery config")

	var cfg struct {
		Options map[string]any `json:"options"`
	}
	err = json.Unmarshal([]byte(generated[configName]), &cfg)
	require.NoError(t, err, "decoding the osquery config")
	return cfg.Options
}
