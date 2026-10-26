// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func publishingInputs(t *testing.T, yaml string) []InputConfig {
	t.Helper()
	c, err := conf.NewConfigWithYAML([]byte(yaml), "")
	require.NoError(t, err, "parsing test YAML")
	var parsed Config
	require.NoError(t, c.Unpack(&parsed), "unpacking test config")
	return parsed.Inputs
}

func TestGetPublishingConfigDefaults(t *testing.T) {
	for name, inputs := range map[string][]InputConfig{
		"no inputs": nil,
		"no options": publishingInputs(t, `
inputs:
  - osquery: {}
`),
	} {
		t.Run(name, func(t *testing.T) {
			got, err := GetPublishingConfig(inputs)
			require.NoError(t, err, "GetPublishingConfig must not fail on defaults")
			assert.Equal(t, 3200, got.Scheduled.MaxEvents, "default scheduled.max_events")
			assert.Equal(t, time.Duration(0), got.Scheduled.MaxAge, "default scheduled.max_age")
			assert.Equal(t, 60*time.Second, got.Live.Timeout, "default live.timeout")
		})
	}
}

func TestGetPublishingConfigOverrides(t *testing.T) {
	inputs := publishingInputs(t, `
inputs:
  - osquery:
      elastic_options:
        publishing:
          scheduled:
            max_events: 100
            max_age: 5m
          live:
            timeout: 10s
`)
	got, err := GetPublishingConfig(inputs)
	require.NoError(t, err, "GetPublishingConfig must accept valid overrides")
	assert.Equal(t, 100, got.Scheduled.MaxEvents, "scheduled.max_events override")
	assert.Equal(t, 5*time.Minute, got.Scheduled.MaxAge, "scheduled.max_age override")
	assert.Equal(t, 10*time.Second, got.Live.Timeout, "live.timeout override")
}

func TestGetPublishingConfigRejectsNegative(t *testing.T) {
	tests := []struct {
		name  string
		yaml  string
		field string
	}{
		{"max_events", "scheduled: {max_events: -1}", "publishing.scheduled.max_events"},
		{"max_age", "scheduled: {max_age: -1s}", "publishing.scheduled.max_age"},
		{"timeout", "live: {timeout: -1s}", "publishing.live.timeout"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			inputs := publishingInputs(t, "inputs:\n  - osquery:\n      elastic_options:\n        publishing: {"+tc.yaml+"}\n")
			_, err := GetPublishingConfig(inputs)
			require.Error(t, err, "negative value must be rejected")
			assert.Contains(t, err.Error(), tc.field, "error must name the offending setting")
		})
	}
}
