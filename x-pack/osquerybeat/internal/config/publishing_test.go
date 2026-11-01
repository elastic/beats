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

func TestPublishingConfigRejectsNegative(t *testing.T) {
	tests := []struct {
		name  string
		yaml  string
		field string
		set   PublishingConfig
	}{
		{"max_events", "scheduled: {max_events: -1}", "publishing.scheduled.max_events", PublishingConfig{Scheduled: ScheduledPublishingConfig{MaxEvents: -1}}},
		{"max_age", "scheduled: {max_age: -1s}", "publishing.scheduled.max_age", PublishingConfig{Scheduled: ScheduledPublishingConfig{MaxAge: -time.Second}}},
		{"timeout", "live: {timeout: -1s}", "publishing.live.timeout", PublishingConfig{Live: LivePublishingConfig{Timeout: -time.Second}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, err := conf.NewConfigWithYAML([]byte("inputs:\n  - osquery:\n      elastic_options:\n        publishing: {"+tc.yaml+"}\n"), "")
			require.NoError(t, err, "parsing test YAML")
			var parsed Config
			err = c.Unpack(&parsed)
			require.Error(t, err, "unpacking a negative value must fail, so startup and policy reloads reject it")
			assert.Contains(t, err.Error(), tc.field, "unpack error must name the offending setting")

			inputs := []InputConfig{{Osquery: &OsqueryConfig{ElasticOptions: &ElasticOptions{Publishing: &tc.set}}}}
			_, err = GetPublishingConfig(inputs)
			require.Error(t, err, "GetPublishingConfig must still reject a negative value that bypassed unpacking")
			assert.Contains(t, err.Error(), tc.field, "error must name the offending setting")
		})
	}
}
