// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchstorage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
)

// TestConfig_Unmarshal_RoutesKnobsAndConnection verifies the real collector
// decode path: the extension-specific knobs land in their typed fields, the
// index sub-section decodes, unset retry sub-fields keep their defaults, and
// everything else (the Elasticsearch connection settings) is captured into the
// ",remain" map rather than being mistaken for a knob.
func TestConfig_Unmarshal_RoutesKnobsAndConnection(t *testing.T) {
	conf := confmap.NewFromStringMap(map[string]any{
		"hosts":    []any{"http://localhost:9200"},
		"username": "elastic",
		"ssl":      map[string]any{"verification_mode": "none"},
		"encoding": "bytes",
		"refresh":  "wait_for",
		"index": map[string]any{
			"number_of_shards":   3,
			"number_of_replicas": 2,
		},
		"retry": map[string]any{
			"max_attempts": 5,
		},
	})

	cfg := createDefaultConfig().(*Config)
	require.NoError(t, conf.Unmarshal(cfg))

	assert.Equal(t, "bytes", cfg.Encoding)
	assert.Equal(t, "wait_for", cfg.Refresh)
	assert.Equal(t, 3, cfg.Index.NumberOfShards)
	assert.Equal(t, 2, cfg.Index.NumberOfReplicas)
	assert.Equal(t, 5, cfg.Retry.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, cfg.Retry.BaseDelay, "unset retry sub-fields keep their defaults")

	assert.Contains(t, cfg.ElasticsearchConfig, "hosts", "connection settings go to the remain map")
	assert.Contains(t, cfg.ElasticsearchConfig, "username")
	assert.Contains(t, cfg.ElasticsearchConfig, "ssl")
	assert.NotContains(t, cfg.ElasticsearchConfig, "encoding", "knobs must not leak into the connection map")
	assert.NotContains(t, cfg.ElasticsearchConfig, "index")
	assert.NotContains(t, cfg.ElasticsearchConfig, "retry")

	require.NoError(t, cfg.Validate())
}

func TestConfig_Validate_RejectsBaseDelayAboveMaxDelay(t *testing.T) {
	c := createDefaultConfig().(*Config)
	c.Retry.BaseDelay = 10 * time.Second // default max_delay is 5s
	assert.Error(t, c.Validate(), "base_delay above max_delay must be rejected")
}

func TestConfig_Validate_RejectsNegativeIndexCounts(t *testing.T) {
	c := createDefaultConfig().(*Config)
	c.Index.NumberOfShards = -1
	assert.Error(t, c.Validate(), "negative shard count must be rejected")

	c = createDefaultConfig().(*Config)
	c.Index.NumberOfReplicas = -1
	assert.Error(t, c.Validate(), "negative replica count must be rejected")
}
