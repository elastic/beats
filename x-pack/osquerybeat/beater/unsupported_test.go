// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beatmonitoring"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	agentconfig "github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
)

// unsupportedCases are policies that use a setting which FIPS builds reject.
// Builds without the requirefips tag accept all of them.
var unsupportedCases = []struct {
	name string
	// osquery is the osquery section of the policy's input.
	osquery map[string]any
	// setting is the setting that FIPS builds name when they reject the policy.
	setting string
}{
	{
		name: "extension_paths",
		osquery: map[string]any{
			"elastic_options": map[string]any{
				"extensions": map[string]any{
					"paths": []any{"/opt/osquery/extensions"},
				},
			},
		},
		setting: "elastic_options.extensions.paths",
	},
	{
		name: "custom_osquery",
		osquery: map[string]any{
			"elastic_options": map[string]any{
				"install": map[string]any{
					"linux": map[string]any{
						"amd64": map[string]any{
							"artifact_url": "https://example.com/osquery.tar.gz",
							"sha256":       "0000000000000000000000000000000000000000000000000000000000000000",
						},
					},
				},
			},
		},
		setting: "elastic_options.install",
	},
	{
		name: "extension_flag",
		osquery: map[string]any{
			"options": map[string]any{
				"extension": "/opt/osquery/extensions/custom.ext",
			},
		},
		setting: "osquery.options.extension",
	},
}

// unsupportedConfig returns the configuration for a beat with one osquery
// input that has the given osquery section.
func unsupportedConfig(osquery map[string]any) *agentconfig.C {
	return agentconfig.MustNewConfigFrom(map[string]any{
		"inputs": []any{
			map[string]any{
				"name":    "osquery-manager-1",
				"type":    "osquery",
				"osquery": osquery,
			},
		},
	})
}

// unsupportedInputs returns the inputs that unsupportedConfig unpacks to, as
// New and the policy watcher read them.
func unsupportedInputs(t *testing.T, osquery map[string]any) []config.InputConfig {
	t.Helper()

	c := config.DefaultConfig
	require.NoError(t, unsupportedConfig(osquery).Unpack(&c), "unpacking the policy")
	return c.Inputs
}

// newWithUnsupportedConfig calls New with the configuration for the given
// osquery section. A beater that New returns is stopped when the test ends.
func newWithUnsupportedConfig(t *testing.T, osquery map[string]any) error {
	t.Helper()

	b := &beat.Beat{
		Info:       beat.Info{Logger: logptest.NewTestingLogger(t, "")},
		Manager:    &testManager{},
		Registry:   reload.NewRegistry(),
		Monitoring: beatmonitoring.NewMonitoring(),
	}
	b.Info.Paths = newTestBeatPaths(t)

	bt, err := New(b, unsupportedConfig(osquery))
	if err == nil {
		t.Cleanup(bt.Stop)
	}
	return err
}
