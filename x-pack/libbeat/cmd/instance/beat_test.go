// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package instance

import (
	"testing"

	"github.com/elastic/beats/v7/filebeat/cmd"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/otelbeat/otelmanager"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.uber.org/zap/zapcore"
)

func TestManager(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := map[string]any{
		"filebeat": map[string]any{
			"inputs": []map[string]any{
				{
					"type":    "benchmark",
					"enabled": true,
					"message": "test",
					"count":   10,
				},
			},
		},
		"output": map[string]any{
			"otelconsumer": map[string]any{},
		},
		"path.home": tmpDir,
	}
	t.Run("otel management disabled - key missing", func(t *testing.T) {
		beat, err := NewBeatForReceiver(cmd.FilebeatSettings("filebeat"), cfg, false, consumertest.NewNop(), "testcomponent", zapcore.NewNopCore())
		assert.NoError(t, err)
		assert.NotNil(t, beat.Manager)
		// it should fallback to FallbackManager if key is missing
		assert.IsType(t, beat.Manager, &management.FallbackManager{})
	})
	t.Run("otel management enabled", func(t *testing.T) {
		cfg["management.otel.enabled"] = true
		beat, err := NewBeatForReceiver(cmd.FilebeatSettings("filebeat"), cfg, false, consumertest.NewNop(), "testcomponent", zapcore.NewNopCore())
		assert.NoError(t, err)
		assert.NotNil(t, beat.Manager)
		assert.IsType(t, beat.Manager, &otelmanager.OtelManager{})
	})
	t.Run("otel management disabled", func(t *testing.T) {
		cfg["management.otel.enabled"] = false
		beat, err := NewBeatForReceiver(cmd.FilebeatSettings("filebeat"), cfg, false, consumertest.NewNop(), "testcomponent", zapcore.NewNopCore())
		assert.NoError(t, err)
		assert.NotNil(t, beat.Manager)
		assert.IsType(t, beat.Manager, &management.FallbackManager{})
	})
}
