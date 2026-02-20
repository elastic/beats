// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbreceiver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

	"github.com/gofrs/uuid/v5"
	"go.opentelemetry.io/collector/pdata/pcommon"

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

	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestNewReceiver(t *testing.T) {
	monitorSocket := genSocketPath(t)
	monitorHost := hostFromSocket(monitorSocket)
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
			assert.Equal(c, "test", logs["r1"][0].Flatten()["message"], "expected message field to contain string 'test'")
			var lastError strings.Builder
			assert.Conditionf(c, func() bool {
				return getFromSocket(t, &lastError, monitorSocket, "stats")
			}, "failed to connect to monitoring socket, stats endpoint, last error was: %s", &lastError)
			assert.Conditionf(c, func() bool {
				return getFromSocket(t, &lastError, monitorSocket, "inputs")
			}, "failed to connect to monitoring socket, inputs endpoint, last error was: %s", &lastError)
			assert.Condition(c, func() bool {
				processorsLoaded := zapLogs.FilterMessageSnippet("Generated new processors")
				assert.Empty(c, processorsLoaded.All(), "processors loaded but none expected")
				// Check that add_host_metadata enrichment is not done.
				return assert.NotContains(c, logs["r1"][0].Flatten(), "host.architecture")
			}, "failed to check processors loaded")
			assert.Condition(c, func() bool {
				metricsStarted := zapLogs.FilterMessageSnippet("Starting metrics logging every 30s")
				return assert.NotEmpty(t, metricsStarted.All(), "metrics logging not started")
			}, "failed to check metrics logging")
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

// multiReceiverConfig creates a Config for testing multiple receivers.
// Each receiver gets a unique home path and a JavaScript processor that loads from its own path.config directory.
func multiReceiverConfig(helper multiReceiverHelper) *Config {
	return &Config{
		Beatconfig: map[string]any{
			"filebeat": map[string]any{
				"inputs": []map[string]any{
					{
						"type":    "benchmark",
						"enabled": true,
						"message": "test",
						"count":   1,
						// Each receiver gets a JavaScript processor that loads from its own
						// path.config directory, adding a unique marker field to verify isolation.
						"processors": []map[string]any{
							{
								"script": map[string]any{
									"lang": "javascript",
									"file": "processor.js",
									"tag":  "js-" + helper.jsMarker,
								},
							},
						},
					},
					{
						"type":                 "filestream",
						"enabled":              true,
						"id":                   "must-be-unique",
						"paths":                []string{helper.ingest},
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
			"path.home":    helper.home,
			"http.enabled": true,
			"http.host":    hostFromSocket(helper.monitorSocket),
		},
	}
}

type multiReceiverHelper struct {
	name          string
	home          string
	ingest        string
	jsMarker      string
	monitorSocket string
}

func newMultiReceiverHelper(t *testing.T, number int) multiReceiverHelper {
	const (
		scriptFormat = `function process(event) { event.Put("js_marker", %q); return event; }`
	)

	home := t.TempDir()

	// Create JavaScript processor files in each receiver's home directory.
	// Each script adds a unique marker field to verify path isolation.
	jsMarker := fmt.Sprintf("receiver%d", number)
	writeFile(t, filepath.Join(home, "processor.js"), fmt.Sprintf(scriptFormat, jsMarker))

	return multiReceiverHelper{
		name:          fmt.Sprintf("r%d", number),
		home:          home,
		ingest:        filepath.Join(t.TempDir(), fmt.Sprintf("test%d.log", number)),
		jsMarker:      jsMarker,
		monitorSocket: genSocketPath(t),
	}
}

// TestMultipleReceivers verifies that multiple receivers can be instantiated in isolation, started, and can ingest logs
// without interfering with each other.
func TestMultipleReceivers(t *testing.T) {
	const nReceivers = 2

	factory := NewFactory()

	helpers := make([]multiReceiverHelper, nReceivers)
	configs := make([]oteltest.ReceiverConfig, nReceivers)
	for i := range helpers {
		helper := newMultiReceiverHelper(t, i)
		helpers[i] = helper
		configs[i] = oteltest.ReceiverConfig{
			Name:    helper.name,
			Beat:    "filebeat",
			Config:  multiReceiverConfig(helper),
			Factory: factory,
		}
	}

	oteltest.CheckReceivers(oteltest.CheckReceiversParams{
		T:           t,
		NumRestarts: 5,
		Receivers:   configs,
		AssertFunc: func(c *assert.CollectT, logs map[string][]mapstr.M, zapLogs *observer.ObservedLogs) {
			allMetaData := make([]string, 0, nReceivers)
			allRegData := make([]string, 0, nReceivers)
			for _, helper := range helpers {
				writeFile(c, helper.ingest, "A log line")

				require.NotEmptyf(c, logs[helper.name], "receiver %v does not have any logs", helper)

				assert.Equalf(c, "test", logs[helper.name][0].Flatten()["message"], "expected %v message field to be 'test'", helper)

				// Verify that each receiver used its own JavaScript processor script.
				// This demonstrates path isolation: each receiver loads processor.js from its own path.config.
				assert.Equalf(c, helper.jsMarker, logs[helper.name][0].Flatten()["js_marker"], "expected %v to have js_marker from its own script", helper)

				// Make sure that each receiver has a separate logger
				// instance and does not interfere with others. Previously, the
				// logger in Beats was global, causing logger fields to be
				// overwritten when multiple receivers started in the same process.
				startLogs := zapLogs.FilterMessageSnippet("Beat ID").FilterField(zap.String("otelcol.component.id", "filebeatreceiver/"+helper.name))
				assert.Equalf(c, 1, startLogs.Len(), "%v should have a single start log", helper)

				startMetricsLogs := zapLogs.FilterMessageSnippet("Starting metrics logging every 30s").FilterField(zap.String("otelcol.component.id", "filebeatreceiver/"+helper.name))
				assert.Equalf(c, 1, startMetricsLogs.Len(), "%v should have a single start metrircs logging every 30s", helper)

				metaPath := filepath.Join(helper.home, "/data/meta.json")
				assert.FileExistsf(c, metaPath, "%s of %v should exist", metaPath, helper)
				metaData, err := os.ReadFile(metaPath)
				assert.NoError(c, err)
				allMetaData = append(allMetaData, string(metaData))

				var lastError strings.Builder
				assert.Conditionf(c, func() bool {
					return getFromSocket(t, &lastError, helper.monitorSocket, "stats")
				}, "failed to connect to monitoring socket of %v, stats endpoint, last error was: %s", helper, &lastError)
				assert.Conditionf(c, func() bool {
					return getFromSocket(t, &lastError, helper.monitorSocket, "inputs")
				}, "failed to connect to monitoring socket of %v, inputs endpoint, last error was: %s", helper, &lastError)

				ingestJson, err := json.Marshal(helper.ingest)
				assert.NoError(c, err)

				regPath := filepath.Join(helper.home, "/data/registry/filebeat/log.json")
				assert.FileExistsf(c, regPath, "receiver %v filebeat registry should exist", helper)
				regData, err := os.ReadFile(regPath)
				allRegData = append(allRegData, string(regData))
				assert.NoError(c, err)
				assert.Containsf(c, string(regData), string(ingestJson), "receiver %v registry should contain '%s', but was: %s", helper, string(ingestJson), string(regData))
			}

			for i := range nReceivers {
				for j := range nReceivers {
					if i == j {
						continue
					}
					h1 := helpers[i]
					h2 := helpers[j]
					assert.NotEqualf(c, allMetaData[i], allMetaData[j], "meta data files between %v and %v should be different", h1, h2)
					assert.NotContainsf(c, allRegData[i], allRegData[j], "receiver %v registry should not contain data from %v registry", h1, h2)
				}
			}
		},
	})
}

func TestReceiverStatus(t *testing.T) {
	benchmarkInputId := "benchmark-id"
	inputStatusAttributes := func(state string, msg string) pcommon.Map {
		eventAttributes := pcommon.NewMap()
		inputStatuses := eventAttributes.PutEmptyMap("inputs")
		benchmarkStatus := inputStatuses.PutEmptyMap(benchmarkInputId)
		benchmarkStatus.PutStr("status", state)
		benchmarkStatus.PutStr("error", msg)
		return eventAttributes
	}
	expectedDegradedErrorMessage := "benchmark input degraded"
	expectedFailedErrorMessage := "benchmark input failed"
	testCases := []struct {
		name            string
		status          *componentstatus.Event
		benchmarkStatus string
	}{
		{
			name: "failed input",
			status: componentstatus.NewEvent(
				componentstatus.StatusPermanentError,
				componentstatus.WithError(errors.New(expectedFailedErrorMessage)),
				componentstatus.WithAttributes(inputStatusAttributes(
					componentstatus.StatusPermanentError.String(), expectedFailedErrorMessage)),
			),
			benchmarkStatus: "failed",
		},
		{
			name: "degraded input",
			status: componentstatus.NewEvent(
				componentstatus.StatusRecoverableError,
				componentstatus.WithError(errors.New(expectedDegradedErrorMessage)),
				componentstatus.WithAttributes(inputStatusAttributes(
					componentstatus.StatusRecoverableError.String(), expectedDegradedErrorMessage)),
			),
			benchmarkStatus: "degraded",
		},
		{
			name: "running input",
			status: componentstatus.NewEvent(componentstatus.StatusOK,
				componentstatus.WithAttributes(inputStatusAttributes(
					componentstatus.StatusOK.String(), ""))),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			config := Config{
				Beatconfig: map[string]any{
					"filebeat": map[string]any{
						"inputs": []map[string]any{
							{
								"id":      benchmarkInputId,
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

func genSocketPath(t *testing.T) string {
	t.Helper()
	socketName, err := uuid.NewV4()
	require.NoError(t, err)
	// Use os.TempDir() for short Unix socket paths
	sockPath := filepath.Join(os.TempDir(), socketName.String()+".sock")
	t.Cleanup(func() { _ = os.Remove(sockPath) })
	return sockPath
}

func getFromSocket(t *testing.T, sb *strings.Builder, socketPath string, endpoint string) bool {
	// skip windows for now
	if runtime.GOOS == "windows" {
		return true
	}
	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
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
	sequenceNum int64
	currentFile string
}

func newLogGenerator(t *testing.T, tmpDir string) *logGenerator {
	return &logGenerator{
		t:      t,
		tmpDir: tmpDir,
	}
}

func (g *logGenerator) Start() {
	if g.currentFile != "" {
		os.Remove(g.currentFile)
	}

	filePath := filepath.Join(g.tmpDir, "input.log")

	f, err := os.Create(filePath)
	require.NoError(g.t, err)
	g.f = f
	g.currentFile = filePath
	atomic.StoreInt64(&g.sequenceNum, 0)
}

func (g *logGenerator) Stop() {
	if g.f != nil {
		require.NoError(g.t, g.f.Close())
		g.f = nil
	}
	if g.currentFile != "" {
		os.Remove(g.currentFile)
		g.currentFile = ""
	}
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
							filepath.Join(tmpDir, "input.log"),
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

func hostFromSocket(socket string) string {
	if runtime.GOOS == "windows" {
		return "npipe:///" + filepath.Base(socket)
	}
	return "unix://" + socket
}

func writeFile(t require.TestingT, path string, data string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	require.NoErrorf(t, err, "Could not open file %s", path)
	defer f.Close()
	_, err = f.WriteString(data + "\n")
	require.NoErrorf(t, err, "Could not write %s to file %s", data, path)
}
