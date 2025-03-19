// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbreceiver

import (
	"bytes"
	"context"
	"testing"

	"github.com/elastic/beats/v7/libbeat/otelbeat/oteltest"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewReceiver(t *testing.T) {
	config := Config{
		Beatconfig: map[string]interface{}{
			"filebeat": map[string]interface{}{
				"inputs": []map[string]interface{}{
					{
						"type":    "benchmark",
						"enabled": true,
						"message": "test",
						"count":   1,
					},
				},
			},
			"output": map[string]interface{}{
				"otelconsumer": map[string]interface{}{},
			},
			"logging": map[string]interface{}{
				"level": "debug",
				"selectors": []string{
					"*",
				},
			},
			"path.home": t.TempDir(),
		},
	}

	oteltest.CheckReceivers(oteltest.CheckReceiversParams{
		T: t,
		Receivers: []oteltest.ReceiverConfig{
			{
				Name:    "r1",
				Config:  &config,
				Factory: NewFactory(),
			},
		},
		AssertFunc: func(t *assert.CollectT, logs map[string][]mapstr.M, zapLogs *observer.ObservedLogs) {
			_ = zapLogs
			assert.Len(t, logs["r1"], 1)
		},
	})
}

func TestReceiverDefaultProcessors(t *testing.T) {
	config := Config{
		Beatconfig: map[string]interface{}{
			"filebeat": map[string]interface{}{
				"inputs": []map[string]interface{}{
					{
						"type":    "benchmark",
						"enabled": true,
						"message": "test",
						"count":   1,
					},
				},
			},
			"output": map[string]interface{}{
				"otelconsumer": map[string]interface{}{},
			},
			"logging": map[string]interface{}{
				"level": "debug",
				"selectors": []string{
					"*",
				},
			},
			"path.home": t.TempDir(),
		},
	}

	oteltest.CheckReceivers(oteltest.CheckReceiversParams{
		T: t,
		Receivers: []oteltest.ReceiverConfig{
			{
				Name:    "r1",
				Config:  &config,
				Factory: NewFactory(),
			},
		},
		AssertFunc: func(t *assert.CollectT, logs map[string][]mapstr.M, zapLogs *observer.ObservedLogs) {
			require.Len(t, logs["r1"], 1)

			processorsLoaded := zapLogs.FilterMessageSnippet("Generated new processors").
				FilterMessageSnippet("add_host_metadata").
				FilterMessageSnippet("add_cloud_metadata").
				FilterMessageSnippet("add_docker_metadata").
				FilterMessageSnippet("add_kubernetes_metadata").
				Len() == 1
			require.True(t, processorsLoaded, "processors not loaded")
			// Check that add_host_metadata works, other processors are not guaranteed to add fields in all environments
			require.Contains(t, logs["r1"][0].Flatten(), "host.architecture")
		},
	})
}

func BenchmarkFactory(b *testing.B) {
	tmpDir := b.TempDir()

	cfg := &Config{
		Beatconfig: map[string]interface{}{
			"filebeat": map[string]interface{}{
				"inputs": []map[string]interface{}{
					{
						"type":    "benchmark",
						"enabled": true,
						"message": "test",
						"count":   10,
					},
				},
			},
			"output": map[string]interface{}{
				"otelconsumer": map[string]interface{}{},
			},
			"logging": map[string]interface{}{
				"level": "debug",
				"selectors": []string{
					"*",
				},
			},
			"path.home": tmpDir,
		},
	}

	var zapLogs bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.Lock(zapcore.AddSync(&zapLogs)),
		zapcore.DebugLevel)

	receiverSettings := receiver.Settings{}
	receiverSettings.Logger = zap.New(core)

	factory := NewFactory()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := factory.CreateLogs(context.Background(), receiverSettings, cfg, nil)
		require.NoError(b, err)
	}
}

func TestMultipleReceivers(t *testing.T) {
	config := Config{
		Beatconfig: map[string]interface{}{
			"filebeat": map[string]interface{}{
				"inputs": []map[string]interface{}{
					{
						"type":    "benchmark",
						"enabled": true,
						"message": "test",
						"count":   1,
					},
				},
			},
			"output": map[string]interface{}{
				"otelconsumer": map[string]interface{}{},
			},
			"logging": map[string]interface{}{
				"level": "debug",
				"selectors": []string{
					"*",
				},
			},
			"path.home": t.TempDir(),
		},
	}

	factory := NewFactory()
	oteltest.CheckReceivers(oteltest.CheckReceiversParams{
		T: t,
		Receivers: []oteltest.ReceiverConfig{
			{
				Name:    "r1",
				Config:  &config,
				Factory: factory,
			},
			{
				Name:    "r2",
				Config:  &config,
				Factory: factory,
			},
		},
		AssertFunc: func(t *assert.CollectT, logs map[string][]mapstr.M, zapLogs *observer.ObservedLogs) {
			_ = zapLogs
			require.Len(t, logs["r1"], 1)
			require.Len(t, logs["r2"], 1)
		},
	})
}

func TestUniqueLoggerNamePerReceiver(t *testing.T) {
	config := Config{
		Beatconfig: map[string]interface{}{
			"filebeat": map[string]interface{}{
				"inputs": []map[string]interface{}{
					{
						"type":    "benchmark",
						"enabled": true,
						"message": "test",
						"count":   1,
					},
				},
			},
			"output": map[string]interface{}{
				"otelconsumer": map[string]interface{}{},
			},
			"logging": map[string]interface{}{
				"level": "debug",
				"selectors": []string{
					"*",
				},
			},
			"path.home": t.TempDir(),
		},
	}

        factory := NewFactory()
	oteltest.CheckReceivers(oteltest.CheckReceiversParams{
		T: t,
		Receivers: []oteltest.ReceiverConfig{
			{
				Name:    "r1",
				Config:  &config,
				Factory: factory,
			},
			{
				Name:    "r2",
				Config:  &config,
				Factory: factory,
			},
		},
		AssertFunc: func(t *assert.CollectT, logs map[string][]mapstr.M, zapLogs *observer.ObservedLogs) {
			require.Conditionf(t, func() bool {
				return len(logs["r1"]) > 0 && len(logs["r2"]) > 0
			}, "expected receivers to have logs, got logs: %v", logs)

			r1StartLogs := zapLogs.FilterMessageSnippet("Starting input").FilterField(zap.String("name", "r1"))
			assert.Equal(t, 1, r1StartLogs.Len(), "r1 should have a single start log")

			r2StartLogs := zapLogs.FilterMessageSnippet("Starting input").FilterField(zap.String("name", "r2"))
			require.Equal(t, 1, r2StartLogs.Len(), "r2 should have a single start log")
		},
	})
}
