// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build requirefips

package config

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The settings that CheckFIPS can name.
const (
	settingPaths     = "elastic_options.extensions.paths"
	settingInstall   = "elastic_options.install"
	settingExtension = "osquery.options.extension"
)

var allUnsupported = []string{settingPaths, settingInstall, settingExtension}

func TestCheckFIPS(t *testing.T) {
	tests := []struct {
		name   string
		inputs []InputConfig
		// unsupported are the settings that the error must name. Settings that
		// are not listed must not be named.
		unsupported []string
	}{
		{name: "no_inputs"},
		{name: "no_osquery_config", inputs: []InputConfig{{}}},
		{
			name: "extension_paths",
			inputs: inputsWith(&OsqueryConfig{ElasticOptions: &ElasticOptions{
				Extensions: &ExtensionsConfig{Paths: []string{"/opt/osquery/extensions"}},
			}}),
			unsupported: []string{settingPaths},
		},
		{
			name: "extension_timeout_and_require_only",
			inputs: inputsWith(&OsqueryConfig{ElasticOptions: &ElasticOptions{
				Extensions: &ExtensionsConfig{Timeout: 30, Require: []string{"ext"}},
			}}),
		},
		{
			name: "extension_empty_paths",
			inputs: inputsWith(&OsqueryConfig{ElasticOptions: &ElasticOptions{
				Extensions: &ExtensionsConfig{Paths: []string{}},
			}}),
		},
		{
			name: "custom_osquery",
			inputs: inputsWith(&OsqueryConfig{ElasticOptions: &ElasticOptions{
				Install: customOsquery(),
			}}),
			unsupported: []string{settingInstall},
		},
		{
			name: "custom_osquery_for_another_platform",
			inputs: inputsWith(&OsqueryConfig{ElasticOptions: &ElasticOptions{
				Install: &InstallConfig{
					Windows: &InstallPlatformConfig{
						ARM64: &InstallArtifactConfig{
							ArtifactURL: "https://example.com/osquery.zip",
							SHA256:      "0000000000000000000000000000000000000000000000000000000000000000",
						},
					},
				},
			}}),
			unsupported: []string{settingInstall},
		},
		{
			name: "install_without_artifact",
			inputs: inputsWith(&OsqueryConfig{ElasticOptions: &ElasticOptions{
				Install: &InstallConfig{Linux: &InstallPlatformConfig{}},
			}}),
		},
		{
			name: "extension_flag",
			inputs: inputsWith(&OsqueryConfig{
				Options: map[string]any{"extension": "/tmp/evil.ext"},
			}),
			unsupported: []string{settingExtension},
		},
		{
			name: "other_options_only",
			inputs: inputsWith(&OsqueryConfig{
				Options: map[string]any{"schedule_splay_percent": 10, "extensions_require": "ext"},
			}),
		},
		{
			name: "all_three",
			inputs: inputsWith(&OsqueryConfig{
				Options: map[string]any{"extension": "/tmp/evil.ext"},
				ElasticOptions: &ElasticOptions{
					Install:    customOsquery(),
					Extensions: &ExtensionsConfig{Paths: []string{"/opt/osquery/extensions"}},
				},
			}),
			unsupported: []string{settingPaths, settingInstall, settingExtension},
		},
		{
			name: "setting_on_an_input_that_is_not_the_result_input",
			inputs: []InputConfig{
				{
					Name:       "results",
					Datastream: DatastreamConfig{Dataset: DefaultDataset},
					Osquery:    &OsqueryConfig{},
				},
				{
					Name:       "other",
					Datastream: DatastreamConfig{Dataset: "something_else"},
					Osquery: &OsqueryConfig{
						Options: map[string]any{"extension": "/tmp/evil.ext"},
					},
				},
			},
			unsupported: []string{settingExtension},
		},
		{
			name: "the_same_setting_on_two_inputs_is_named_once",
			inputs: append(
				inputsWith(&OsqueryConfig{Options: map[string]any{"extension": "/tmp/a.ext"}}),
				inputsWith(&OsqueryConfig{Options: map[string]any{"extension": "/tmp/b.ext"}})...,
			),
			unsupported: []string{settingExtension},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckFIPS(tc.inputs)

			if len(tc.unsupported) == 0 {
				assert.NoError(t, err, "settings that FIPS builds support must be accepted")
				return
			}
			require.Error(t, err, "FIPS builds must reject %v", tc.unsupported)
			assert.ErrorContains(t, err, "not supported in FIPS builds", "the error must say why the policy is rejected")
			assert.NotContains(t, err.Error(), "\n", "the error must be a single line")
			for _, setting := range allUnsupported {
				if slices.Contains(tc.unsupported, setting) {
					assert.ErrorContains(t, err, setting, "the error must name %s", setting)
					assert.Equal(t, 1, strings.Count(err.Error(), setting+" is not supported"), "the error must name %s once", setting)
				} else {
					assert.NotContains(t, err.Error(), setting, "the error must not name %s, which the policy does not use", setting)
				}
			}
		})
	}
}

func inputsWith(osquery *OsqueryConfig) []InputConfig {
	return []InputConfig{{
		Name:    "osquery-manager-1",
		Type:    "osquery",
		Osquery: osquery,
	}}
}

func customOsquery() *InstallConfig {
	return &InstallConfig{
		Linux: &InstallPlatformConfig{
			AMD64: &InstallArtifactConfig{
				ArtifactURL: "https://example.com/osquery.tar.gz",
				SHA256:      "0000000000000000000000000000000000000000000000000000000000000000",
			},
		},
	}
}
