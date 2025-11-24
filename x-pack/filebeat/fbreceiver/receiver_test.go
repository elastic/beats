// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbreceiver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
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
	monitorSocket := genSocketPath()
	var monitorHost string
	if runtime.GOOS == "windows" {
		monitorHost = "npipe:///" + filepath.Base(monitorSocket)
	} else {
		monitorHost = "unix://" + monitorSocket
	}
	config := Config{
		Beatconfig: map[string]any{
			"filebeat": map[string]any{
				"inputs": []map[string]any{
					{
						"type":    "benchmark",
						"enabled": true,
						"message": "test",
						"count":   1,
					},
					{
						"type":    "filestream",
						"enabled": true,
						"id":      "must-be-unique",
						"paths":   []string{"none"},
					},
				},
			},
			"logging": map[string]any{
				"level": "debug",
				"selectors": []string{
					"*",
				},
			},
			"path.home":               t.TempDir(),
			"http.enabled":            true,
			"http.host":               monitorHost,
			"management.otel.enabled": true,
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
		AssertFunc: func(c *assert.CollectT, logs map[string][]mapstr.M, zapLogs *observer.ObservedLogs) {
			_ = zapLogs
			require.Lenf(c, logs["r1"], 1, "expected 1 log, got %d", len(logs["r1"]))
			assert.Equal(c, "filebeatreceiver/r1", logs["r1"][0].Flatten()["agent.otelcol.component.id"], "expected agent.otelcol.component.id field in log record")
			assert.Equal(c, "receiver", logs["r1"][0].Flatten()["agent.otelcol.component.kind"], "expected agent.otelcol.component.kind field in log record")
			var lastError strings.Builder
			assert.Conditionf(c, func() bool {
				return getFromSocket(t, &lastError, monitorSocket, "stats")
			}, "failed to connect to monitoring socket, stats endpoint, last error was: %s", &lastError)
			assert.Conditionf(c, func() bool {
				return getFromSocket(t, &lastError, monitorSocket, "inputs")
			}, "failed to connect to monitoring socket, inputs endpoint, last error was: %s", &lastError)
			assert.Condition(c, func() bool {
				processorsLoaded := zapLogs.FilterMessageSnippet("Generated new processors").
					FilterMessageSnippet("add_host_metadata").
					FilterMessageSnippet("add_cloud_metadata").
					FilterMessageSnippet("add_docker_metadata").
					FilterMessageSnippet("add_kubernetes_metadata")
				assert.Len(t, processorsLoaded.All(), 1, "processors not loaded")
				// Check that add_host_metadata works, other processors are not guaranteed to add fields in all environments
				return assert.Contains(c, logs["r1"][0].Flatten(), "host.architecture")
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
		Beatconfig: map[string]any{
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
			"logging": map[string]any{
				"level": level.String(),
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
	for b.Loop() {
		_, err := factory.CreateLogs(b.Context(), receiverSettings, cfg, nil)
		require.NoError(b, err)
	}
}

func TestMultipleReceivers(t *testing.T) {
	// This test verifies that multiple receivers can be instantiated
	// in isolation, started, and can ingest logs without interfering
	// with each other.

	// Receivers need distinct home directories so wrap the config in a function.
	config := func(monitorSocket string, homePath string, ingestPath string) *Config {
		var monitorHost string
		if runtime.GOOS == "windows" {
			monitorHost = "npipe:///" + filepath.Base(monitorSocket)
		} else {
			monitorHost = "unix://" + monitorSocket
		}
		return &Config{
			Beatconfig: map[string]any{
				"filebeat": map[string]any{
					"inputs": []map[string]any{
						{
							"type":    "benchmark",
							"enabled": true,
							"message": "test",
							"count":   1,
						},
						{
							"type":                 "filestream",
							"enabled":              true,
							"id":                   "must-be-unique",
							"paths":                []string{ingestPath},
							"file_identity.native": nil,
						},
					},
				},
				"logging": map[string]any{
					"level": "info",
					"selectors": []string{
						"*",
					},
				},
				"path.home":    homePath,
				"http.enabled": true,
				"http.host":    monitorHost,
			},
		}
	}

	factory := NewFactory()
	monitorSocket1 := genSocketPath()
	monitorSocket2 := genSocketPath()
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	ingest1 := filepath.Join(t.TempDir(), "test1.log")
	ingest2 := filepath.Join(t.TempDir(), "test2.log")
	oteltest.CheckReceivers(oteltest.CheckReceiversParams{
		T:           t,
		NumRestarts: 5,
		Receivers: []oteltest.ReceiverConfig{
			{
				Name:    "r1",
				Beat:    "filebeat",
				Config:  config(monitorSocket1, dir1, ingest1),
				Factory: factory,
			},
			{
				Name:    "r2",
				Beat:    "filebeat",
				Config:  config(monitorSocket2, dir2, ingest2),
				Factory: factory,
			},
		},
		AssertFunc: func(c *assert.CollectT, logs map[string][]mapstr.M, zapLogs *observer.ObservedLogs) {
			// Add data to be ingested with filestream
			f1, err := os.OpenFile(ingest1, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			require.NoError(c, err)
			_, err = f1.WriteString("A log line\n")
			require.NoError(c, err)
			f1.Close()
			f2, err := os.OpenFile(ingest2, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			require.NoError(c, err)
			_, err = f2.WriteString("A log line\n")
			require.NoError(c, err)
			f2.Close()

			require.Greater(c, len(logs["r1"]), 0, "receiver r1 does not have any logs")
			require.Greater(c, len(logs["r2"]), 0, "receiver r2 does not have any logs")

			assert.Equal(c, "filebeatreceiver/r1", logs["r1"][0].Flatten()["agent.otelcol.component.id"], "expected agent.otelcol.component.id field in r1 log record")
			assert.Equal(c, "receiver", logs["r1"][0].Flatten()["agent.otelcol.component.kind"], "expected agent.otelcol.component.kind field in r1 log record")
			assert.Equal(c, "filebeatreceiver/r2", logs["r2"][0].Flatten()["agent.otelcol.component.id"], "expected agent.otelcol.component.id field in r2 log record")
			assert.Equal(c, "receiver", logs["r2"][0].Flatten()["agent.otelcol.component.kind"], "expected agent.otelcol.component.kind field in r2 log record")

			// Make sure that each receiver has a separate logger
			// instance and does not interfere with others. Previously, the
			// logger in Beats was global, causing logger fields to be
			// overwritten when multiple receivers started in the same process.
			r1StartLogs := zapLogs.FilterMessageSnippet("Beat ID").FilterField(zap.String("otelcol.component.id", "filebeatreceiver/r1"))
			assert.Equal(c, 1, r1StartLogs.Len(), "r1 should have a single start log")
			r2StartLogs := zapLogs.FilterMessageSnippet("Beat ID").FilterField(zap.String("otelcol.component.id", "filebeatreceiver/r2"))
			assert.Equal(c, 1, r2StartLogs.Len(), "r2 should have a single start log")

			meta1Path := filepath.Join(dir1, "/data/meta.json")
			assert.FileExists(c, meta1Path, "dir1/data/meta.json should exist")
			meta1Data, err := os.ReadFile(meta1Path)
			assert.NoError(c, err)

			meta2Path := filepath.Join(dir2, "/data/meta.json")
			assert.FileExists(c, meta2Path, "dir2/data/meta.json should exist")
			meta2Data, err := os.ReadFile(meta2Path)
			assert.NoError(c, err)

			assert.NotEqual(c, meta1Data, meta2Data, "meta data files should be different")

			var lastError strings.Builder
			assert.Conditionf(c, func() bool {
				return getFromSocket(t, &lastError, monitorSocket1, "stats")
			}, "failed to connect to monitoring socket1, stats endpoint, last error was: %s", &lastError)
			assert.Conditionf(c, func() bool {
				return getFromSocket(t, &lastError, monitorSocket1, "inputs")
			}, "failed to connect to monitoring socket1, inputs endpoint, last error was: %s", &lastError)
			assert.Conditionf(c, func() bool {
				return getFromSocket(t, &lastError, monitorSocket2, "stats")
			}, "failed to connect to monitoring socket2, stats endpoint, last error was: %s", &lastError)
			assert.Conditionf(c, func() bool {
				return getFromSocket(t, &lastError, monitorSocket2, "inputs")
			}, "failed to connect to monitoring socket2, inputs endpoint, last error was: %s", &lastError)

			ingest1Json, err := json.Marshal(ingest1)
			require.NoError(c, err)
			ingest2Json, err := json.Marshal(ingest2)
			require.NoError(c, err)

			reg1Path := filepath.Join(dir1, "/data/registry/filebeat/log.json")
			require.FileExists(c, reg1Path, "receiver 1 filebeat registry should exist")
			reg1Data, err := os.ReadFile(reg1Path)
			require.NoError(c, err)
			require.Containsf(c, string(reg1Data), string(ingest1Json), "receiver 1 registry should contain '%s', but was: %s", string(ingest1Json), string(reg1Data))
			require.NotContainsf(c, string(reg1Data), string(ingest2Json), "receiver 1 registry should not contain '%s', but was: %s", string(ingest2Json), string(reg1Data))

			reg2Path := filepath.Join(dir2, "/data/registry/filebeat/log.json")
			require.FileExists(c, reg2Path, "receiver 2 filebeat registry should exist")
			reg2Data, err := os.ReadFile(reg2Path)
			require.NoError(c, err)
			require.Containsf(c, string(reg2Data), string(ingest2Json), "receiver 2 registry should contain '%s', but was: %s", string(ingest2Json), string(reg2Data))
			require.NotContainsf(c, string(reg2Data), string(ingest1Json), "receiver 2 registry should not contain '%s', but was: %s", string(ingest1Json), string(reg2Data))
		},
	})
}

func TestReceiverDegraded(t *testing.T) {
	if runtime.GOARCH == "arm64" && runtime.GOOS == "linux" {
		t.Skip("flaky test on Ubuntu arm64, see https://github.com/elastic/beats/issues/46437")
	}
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

func genSocketPath() string {
	randData := make([]byte, 16)
	for i := range len(randData) {
		randData[i] = uint8(rand.UintN(255)) //nolint:gosec // 0-255 fits in a uint8
	}
	socketName := base64.URLEncoding.EncodeToString(randData) + ".sock"
	socketDir := os.TempDir()
	return filepath.Join(socketDir, socketName)
}

func getFromSocket(t *testing.T, sb *strings.Builder, socketPath string, endpoint string) bool {
	// skip windows for now
	if runtime.GOOS == "windows" {
		return true
	}
	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}
	defer client.CloseIdleConnections()
	url, err := url.JoinPath("http://unix", endpoint)
	if err != nil {
		sb.Reset()
		fmt.Fprintf(sb, "JoinPath failed: %s", err)
		return false
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		sb.Reset()
		fmt.Fprintf(sb, "error creating request: %s", err)
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		sb.Reset()
		fmt.Fprintf(sb, "client.Get failed: %s", err)
		return false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		sb.Reset()
		fmt.Fprintf(sb, "io.ReadAll of body failed: %s", err)
		return false
	}
	if len(body) <= 0 {
		sb.Reset()
		sb.WriteString("body too short")
		return false
	}

	if endpoint == "inputs" {
		var data []any
		if err := json.Unmarshal(body, &data); err != nil {
			sb.Reset()
			fmt.Fprintf(sb, "json unmarshal of body failed: %s\n", err)
			fmt.Fprintf(sb, "body was %v\n", body)
			return false
		}

		if len(data) <= 0 {
			sb.Reset()
			sb.WriteString("json array didn't have any entries")
			return false
		}
	} else {
		var data map[string]any
		if err := json.Unmarshal(body, &data); err != nil {
			sb.Reset()
			fmt.Fprintf(sb, "json unmarshal of body failed: %s\n", err)
			fmt.Fprintf(sb, "body was %v\n", body)
			return false
		}

		if len(data) <= 0 {
			sb.Reset()
			sb.WriteString("json didn't have any keys")
			return false
		}
	}

	return true
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

	defer oteltest.VerifyNoLeaks(t)

	tmpDir := t.TempDir()
	const logsPerTest = 100

	gen := newLogGenerator(t, tmpDir)

	t.Setenv("OTELCONSUMER_RECEIVERTEST", "1")

	cfg := &Config{
		Beatconfig: map[string]any{
			"queue.mem.flush.timeout": "0s",
			"filebeat": map[string]any{
				"inputs": []map[string]any{
					{
						"type":    "filestream",
						"id":      "filestream-test",
						"enabled": true,
						"paths": []string{
							filepath.Join(tmpDir, "input-*.log"),
						},
						"file_identity.native": map[string]any{},
						"prospector": map[string]any{
							"scanner": map[string]any{
								"fingerprint.enabled": false,
								"check_interval":      "0.1s",
							},
						},
						"parsers": []map[string]any{
							{
								"ndjson": map[string]any{
									"document_id": "id",
								},
							},
						},
					},
				},
			},
			"logging": map[string]any{
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

func TestReceiverHook(t *testing.T) {
	cfg := Config{
		Beatconfig: map[string]any{
			"filebeat": map[string]any{
				"inputs": []map[string]any{
					{
						"type":    "benchmark",
						"enabled": true,
						"message": "test",
						"count":   1,
					},
				},
			},
			"management.otel.enabled": true,
			"path.home":               t.TempDir(),
		},
	}
	receiverSettings := receiver.Settings{
		ID: component.MustNewID(Name),
		TelemetrySettings: component.TelemetrySettings{
			Logger: zap.NewNop(),
		},
	}
	// For filebeatreceiver, we expect 3 hooks to be registered:
	// 	one for beat metrics, one for input metrics and one for getting the registry.
	oteltest.TestReceiverHook(t, &cfg, NewFactory(), receiverSettings, 3)
}
