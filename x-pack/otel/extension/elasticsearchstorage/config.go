// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchstorage

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
)

// Config holds the extension configuration. The Elasticsearch connection
// settings (hosts, credentials, TLS, ...) are captured into
// ElasticsearchConfig via mapstructure's ",remain"; the typed fields are
// all optional.
type Config struct {
	ElasticsearchConfig map[string]any `mapstructure:",remain"`

	// Encoding controls how values are stored. "auto" (default) inspects
	// each value and stores valid JSON verbatim (enc:json) and anything
	// else base64-wrapped (enc:base64). "json" pins JSON and skips the
	// validity scan (caller guarantees JSON). "bytes" always base64-wraps.
	Encoding string `mapstructure:"encoding"`

	// Refresh maps to the Elasticsearch write ?refresh parameter. Empty
	// (default) does not force a refresh; "wait_for" or "true" enable
	// read-after-write visibility at a latency cost.
	Refresh string `mapstructure:"refresh"`

	// Retry bounds transient-failure retries around each ES request.
	Retry RetryConfig `mapstructure:"retry"`

	// Index tunes the storage index layout (shard/replica counts).
	Index IndexConfig `mapstructure:"index"`
}

// IndexConfig tunes the storage index layout. These settings apply only to
// stateful clusters: Elastic Cloud Serverless manages shards and replicas
// itself and rejects them on create, so they are omitted there regardless of
// this config. A value of 0 for NumberOfShards means "use the default".
type IndexConfig struct {
	NumberOfShards   int `mapstructure:"number_of_shards"`   // default 1
	NumberOfReplicas int `mapstructure:"number_of_replicas"` // default 0
}

// RetryConfig tunes the bounded, backing-off retry applied to transient
// Elasticsearch failures (429/502/503/504 and network errors). Permanent
// responses (400/401/403/404/409) are never retried.
type RetryConfig struct {
	MaxAttempts int           `mapstructure:"max_attempts"` // default 3
	BaseDelay   time.Duration `mapstructure:"base_delay"`   // default 100ms
	MaxDelay    time.Duration `mapstructure:"max_delay"`    // default 5s
}

// createDefaultConfig returns the config with the extension-specific values
// populated with their defaults.
func createDefaultConfig() component.Config {
	return &Config{
		Encoding: "auto",
		Retry: RetryConfig{
			MaxAttempts: 3,
			BaseDelay:   100 * time.Millisecond,
			MaxDelay:    5 * time.Second,
		},
		Index: IndexConfig{
			NumberOfShards:   1,
			NumberOfReplicas: 0,
		},
	}
}

func (c *Config) Validate() error {
	switch c.Encoding {
	case "", "auto", "json", "bytes":
	default:
		return fmt.Errorf(`elasticsearch_storage: invalid encoding %q (want "auto", "json", or "bytes")`, c.Encoding)
	}

	switch c.Refresh {
	case "", "false", "true", "wait_for":
	default:
		return fmt.Errorf(`elasticsearch_storage: invalid refresh %q (want "true", "false", or "wait_for")`, c.Refresh)
	}

	if c.Retry.MaxAttempts < 0 {
		return fmt.Errorf("elasticsearch_storage: retry.max_attempts must be >= 0, got %d", c.Retry.MaxAttempts)
	}
	if c.Retry.BaseDelay < 0 {
		return fmt.Errorf("elasticsearch_storage: retry.base_delay must be >= 0, got %s", c.Retry.BaseDelay)
	}
	if c.Retry.MaxDelay < 0 {
		return fmt.Errorf("elasticsearch_storage: retry.max_delay must be >= 0, got %s", c.Retry.MaxDelay)
	}
	if c.Retry.MaxDelay > 0 && c.Retry.BaseDelay > c.Retry.MaxDelay {
		return fmt.Errorf("elasticsearch_storage: retry.base_delay (%s) must not exceed retry.max_delay (%s)", c.Retry.BaseDelay, c.Retry.MaxDelay)
	}

	// 0 means "use the default"; only negatives are invalid.
	if c.Index.NumberOfShards < 0 {
		return fmt.Errorf("elasticsearch_storage: index.number_of_shards must be >= 0, got %d", c.Index.NumberOfShards)
	}
	if c.Index.NumberOfReplicas < 0 {
		return fmt.Errorf("elasticsearch_storage: index.number_of_replicas must be >= 0, got %d", c.Index.NumberOfReplicas)
	}
	return nil
}
