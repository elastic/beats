// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package instance

import (
	"maps"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/v7/filebeat/cmd"
	"github.com/elastic/beats/v7/filebeat/input/log"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/x-pack/otel/otelmanager"
	conf "github.com/elastic/elastic-agent-libs/config"
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

		// test if log input is enabled
		cfg, err := conf.NewConfigFrom(`
type: "log"`)
		require.NoError(t, err)
		assert.True(t, log.AllowDeprecatedUse(cfg))
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

		// test if log input is disabled
		cfg, err := conf.NewConfigFrom(`
type: "log"`)
		require.NoError(t, err)
		assert.False(t, log.AllowDeprecatedUse(cfg))
	})
}

func TestNewBeatForReceiverMetricLoggingDefault(t *testing.T) {
	tmpDir := t.TempDir()
	baseCfg := map[string]any{
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

	t.Run("defaults to disabled metric logging when unset", func(t *testing.T) {
		beat, err := NewBeatForReceiver(cmd.FilebeatSettings("filebeat"), baseCfg, consumertest.NewNop(), "testcomponent", zapcore.NewNopCore())
		require.NoError(t, err)
		require.NotNil(t, beat.Config.MetricLogging)

		var metricCfg struct {
			Period time.Duration `config:"period"`
		}
		require.NoError(t, beat.Config.MetricLogging.Unpack(&metricCfg))
		assert.Equal(t, time.Duration(0), metricCfg.Period)
	})

	t.Run("honors an explicitly configured period", func(t *testing.T) {
		tmpCfg := map[string]any{}
		maps.Copy(tmpCfg, baseCfg)
		tmpCfg["logging.metrics.period"] = "15s"

		beat, err := NewBeatForReceiver(cmd.FilebeatSettings("filebeat"), tmpCfg, consumertest.NewNop(), "testcomponent", zapcore.NewNopCore())
		require.NoError(t, err)
		require.NotNil(t, beat.Config.MetricLogging)

		var metricCfg struct {
			Period time.Duration `config:"period"`
		}
		require.NoError(t, beat.Config.MetricLogging.Unpack(&metricCfg))
		assert.Equal(t, 15*time.Second, metricCfg.Period)
	})

	t.Run("keeps the default period when an unrelated metric logging field is set", func(t *testing.T) {
		tmpCfg := map[string]any{}
		maps.Copy(tmpCfg, baseCfg)
		tmpCfg["logging.metrics.enabled"] = true

		beat, err := NewBeatForReceiver(cmd.FilebeatSettings("filebeat"), tmpCfg, consumertest.NewNop(), "testcomponent", zapcore.NewNopCore())
		require.NoError(t, err)
		require.NotNil(t, beat.Config.MetricLogging)

		var metricCfg struct {
			Enabled bool          `config:"enabled"`
			Period  time.Duration `config:"period"`
		}
		require.NoError(t, beat.Config.MetricLogging.Unpack(&metricCfg))
		assert.True(t, metricCfg.Enabled)
		assert.Equal(t, time.Duration(0), metricCfg.Period)
	})
}
