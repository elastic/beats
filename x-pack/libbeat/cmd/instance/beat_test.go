// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package instance

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/v7/filebeat/cmd"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/x-pack/otel/otelmanager"
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
		"path.home": tmpDir,
	}
	t.Run("otel management disabled - key missing", func(t *testing.T) {
		beat, err := NewBeatForReceiver(cmd.FilebeatSettings("filebeat"), cfg, consumertest.NewNop(), "testcomponent", zapcore.NewNopCore())
		assert.NoError(t, err)
		assert.NotNil(t, beat.Manager)
		// it should fallback to FallbackManager if key is missing
		assert.IsType(t, &management.FallbackManager{}, beat.Manager)
		assert.False(t, management.UnderAgent())
	})
	t.Run("otel management enabled", func(t *testing.T) {
		tmpCfg := map[string]any{}
		maps.Copy(tmpCfg, cfg)
		tmpCfg["management.otel.enabled"] = true
		defer func() {
			management.SetUnderAgent(false) // reset to false
		}()
		beat, err := NewBeatForReceiver(cmd.FilebeatSettings("filebeat"), tmpCfg, consumertest.NewNop(), "testcomponent", zapcore.NewNopCore())
		assert.NoError(t, err)
		assert.NotNil(t, beat.Manager)
		assert.IsType(t, &otelmanager.OtelManager{}, beat.Manager)
		assert.True(t, management.UnderAgent())
	})
	t.Run("otel management disabled", func(t *testing.T) {
		tmpCfg := map[string]any{}
		maps.Copy(tmpCfg, cfg)
		tmpCfg["management.otel.enabled"] = false
		defer func() {
			management.SetUnderAgent(false) // reset to false
		}()
		beat, err := NewBeatForReceiver(cmd.FilebeatSettings("filebeat"), tmpCfg, consumertest.NewNop(), "testcomponent", zapcore.NewNopCore())
		assert.NoError(t, err)
		assert.NotNil(t, beat.Manager)
		assert.IsType(t, &management.FallbackManager{}, beat.Manager)
		assert.False(t, management.UnderAgent())
	})
}
