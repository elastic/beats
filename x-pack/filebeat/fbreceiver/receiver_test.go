// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbreceiver

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/elastic/beats/v7/libbeat/otelbeat/oteltest"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
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
				Beat:    "filebeat",
				Config:  &config,
				Factory: NewFactory(),
			},
		},
		AssertFunc: func(t *assert.CollectT, logs map[string][]mapstr.M, zapLogs *observer.ObservedLogs) {
			_ = zapLogs
			require.Lenf(t, logs["r1"], 1, "expected 1 log, got %d", len(logs["r1"]))
			assert.Condition(t, func() bool {
				processorsLoaded := zapLogs.FilterMessageSnippet("Generated new processors").
					FilterMessageSnippet("add_host_metadata").
					FilterMessageSnippet("add_cloud_metadata").
					FilterMessageSnippet("add_docker_metadata").
<<<<<<< HEAD
					FilterMessageSnippet("add_kubernetes_metadata").
					Len() == 1
				assert.True(t, processorsLoaded, "processors not loaded")
=======
					FilterMessageSnippet("add_kubernetes_metadata")
				assert.Len(t, processorsLoaded.All(), 1, "processors not loaded")
>>>>>>> 8879ddadf (fix: add_cloud_metadata: Do not block on String() (#47058))
				// Check that add_host_metadata works, other processors are not guaranteed to add fields in all environments
				return assert.Contains(t, logs["r1"][0].Flatten(), "host.architecture")
			}, "failed to check processors loaded")
		},
	})
}

func BenchmarkFactory(b *testing.B) {
	for _, level := range []zapcore.Level{zapcore.InfoLevel, zapcore.DebugLevel} {
		b.Run(level.String(), func(b *testing.B) {
			benchmarkFactoryWithLogLevel(b, level)
		})
	}
}

func benchmarkFactoryWithLogLevel(b *testing.B, level zapcore.Level) {
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
<<<<<<< HEAD
			"logging": map[string]interface{}{
				"level": "info",
=======
			"logging": map[string]any{
				"level": level.String(),
>>>>>>> 8879ddadf (fix: add_cloud_metadata: Do not block on String() (#47058))
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
		level)

	factory := NewFactory()

	receiverSettings := receiver.Settings{}
	receiverSettings.Logger = zap.New(core)
	receiverSettings.ID = component.NewIDWithName(factory.Type(), "r1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := factory.CreateLogs(b.Context(), receiverSettings, cfg, nil)
		require.NoError(b, err)
	}
}

func TestMultipleReceivers(t *testing.T) {
	t.Skip("flaky test, see https://github.com/elastic/beats/issues/43832")
	// This test verifies that multiple receivers can be instantiated
	// in isolation, started, and can ingest logs without interfering
	// with each other.

	// Receivers need distinct home directories so wrap the config in a function.
	config := func() *Config {
		return &Config{
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
					"level": "info",
					"selectors": []string{
						"*",
					},
				},
				"path.home": t.TempDir(),
			},
		}
	}

	factory := NewFactory()
	oteltest.CheckReceivers(oteltest.CheckReceiversParams{
		T: t,
		Receivers: []oteltest.ReceiverConfig{
			{
				Name:    "r1",
				Beat:    "filebeat",
				Config:  config(),
				Factory: factory,
			},
			{
				Name:    "r2",
				Beat:    "filebeat",
				Config:  config(),
				Factory: factory,
			},
		},
		AssertFunc: func(c *assert.CollectT, logs map[string][]mapstr.M, zapLogs *observer.ObservedLogs) {
			require.Greater(c, len(logs["r1"]), 0, "receiver r1 does not have any logs")
			require.Greater(c, len(logs["r2"]), 0, "receiver r2 does not have any logs")

			// Make sure that each receiver has a separate logger
			// instance and does not interfere with others. Previously, the
			// logger in Beats was global, causing logger fields to be
			// overwritten when multiple receivers started in the same process.
			r1StartLogs := zapLogs.FilterMessageSnippet("Beat ID").FilterField(zap.String("otelcol.component.id", "r1"))
			assert.Equal(c, 1, r1StartLogs.Len(), "r1 should have a single start log")
			r2StartLogs := zapLogs.FilterMessageSnippet("Beat ID").FilterField(zap.String("otelcol.component.id", "r2"))
			assert.Equal(c, 1, r2StartLogs.Len(), "r2 should have a single start log")
		},
	})
}

func TestReceiverDegraded(t *testing.T) {
	testCases := []struct {
		name            string
		status          oteltest.ExpectedStatus
		benchmarkStatus string
	}{
		{
			name: "failed input",
			status: oteltest.ExpectedStatus{
				Status: componentstatus.StatusPermanentError,
				Error:  "benchmark input failed",
			},
			benchmarkStatus: "failed",
		},
		{
			name: "degraded input",
			status: oteltest.ExpectedStatus{
				Status: componentstatus.StatusRecoverableError,
				Error:  "benchmark input degraded",
			},
			benchmarkStatus: "degraded",
		},
		{
			name: "running input",
			status: oteltest.ExpectedStatus{
				Status: componentstatus.StatusOK,
				Error:  "",
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			config := Config{
				Beatconfig: map[string]any{
					"filebeat": map[string]any{
						"inputs": []map[string]any{
							{
								"type":    "benchmark",
								"enabled": true,
								"message": "test",
								"count":   1,
								"status":  test.benchmarkStatus,
							},
						},
					},
					"output": map[string]any{
						"otelconsumer": map[string]any{},
					},
					"logging": map[string]any{
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
						Beat:    "filebeat",
						Config:  &config,
						Factory: NewFactory(),
					},
				},
				Status: test.status,
			})
		})
	}
}

type logGenerator struct {
	t           *testing.T
	tmpDir      string
	f           *os.File
	filePattern string
	sequenceNum int64
}

func newLogGenerator(t *testing.T, tmpDir string) *logGenerator {
	return &logGenerator{
		t:           t,
		tmpDir:      tmpDir,
		filePattern: "input-*.log",
	}
}

func (g *logGenerator) Start() {
	f, err := os.CreateTemp(g.tmpDir, g.filePattern)
	require.NoError(g.t, err)
	g.f = f
}

func (g *logGenerator) Stop() {
	require.NoError(g.t, g.f.Close())
}

func (g *logGenerator) Generate() []receivertest.UniqueIDAttrVal {
	id := receivertest.UniqueIDAttrVal(strconv.FormatInt(atomic.AddInt64(&g.sequenceNum, 1), 10))

	_, err := fmt.Fprintln(g.f, `{"id": "`+id+`", "message": "log message"}`)
	require.NoError(g.t, err, "failed to write log line to file")
	require.NoError(g.t, g.f.Sync(), "failed to sync log file")

	return []receivertest.UniqueIDAttrVal{id}
}

// TestConsumeContract tests the ConsumeLogs contract for otelconsumer.
//
// The following scenarios are tested:
// - Always succeed. We expect all data passed to ConsumeLogs to be delivered.
// - Random non-permanent error. We expect the batch to be retried.
// - Random permanent error. We expect the batch to be dropped.
// - Random error. We expect the batch to be retried or dropped based on the error type.
func TestConsumeContract(t *testing.T) {
	t.Skip("flaky test, see https://github.com/elastic/beats/issues/46437")

	tmpDir := t.TempDir()
	const logsPerTest = 100

	gen := newLogGenerator(t, tmpDir)

	t.Setenv("OTELCONSUMER_RECEIVERTEST", "1")

	cfg := &Config{
		Beatconfig: map[string]interface{}{
			"queue.mem.flush.timeout": "0s",
			"filebeat": map[string]interface{}{
				"inputs": []map[string]interface{}{
					{
						"type":    "filestream",
						"id":      "filestream-test",
						"enabled": true,
						"paths": []string{
							filepath.Join(tmpDir, "input-*.log"),
						},
						"file_identity.native": map[string]interface{}{},
						"prospector": map[string]interface{}{
							"scanner": map[string]interface{}{
								"fingerprint.enabled": false,
								"check_interval":      "0.1s",
							},
						},
						"parsers": []map[string]interface{}{
							{
								"ndjson": map[string]interface{}{
									"document_id": "id",
								},
							},
						},
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
			"path.logs": tmpDir,
		},
	}

	// Run the contract checker. This will trigger test failures if any problems are found.
	receivertest.CheckConsumeContract(receivertest.CheckConsumeContractParams{
		T:             t,
		Factory:       NewFactory(),
		Signal:        pipeline.SignalLogs,
		Config:        cfg,
		Generator:     gen,
		GenerateCount: logsPerTest,
	})
}
