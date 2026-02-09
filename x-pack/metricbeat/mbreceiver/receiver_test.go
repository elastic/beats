// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mbreceiver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/receiver"

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
			"metricbeat": map[string]any{
				"modules": []map[string]any{
					{
						"module":     "system",
						"enabled":    true,
						"period":     "1s",
						"processes":  []string{".*"},
						"metricsets": []string{"cpu"},
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
				Beat:    "metricbeat",
				Config:  &config,
				Factory: NewFactory(),
			},
		},
		AssertFunc: func(c *assert.CollectT, logs map[string][]mapstr.M, zapLogs *observer.ObservedLogs) {
			_ = zapLogs
			require.Conditionf(c, func() bool {
				return len(logs["r1"]) > 0
			}, "expected at least one ingest log, got logs: %v", logs["r1"])
			assert.Equal(c, "metricbeat", logs["r1"][0].Flatten()["agent.type"], "expected agent.type field in to be 'metricbeat'")

			var lastError strings.Builder
			assert.Conditionf(c, func() bool {
				return getFromSocket(t, &lastError, monitorSocket, "stats")
			}, "failed to connect to monitoring socket stats endpoint, last error was: %s", &lastError)
			assert.Conditionf(c, func() bool {
				return getFromSocket(t, &lastError, monitorSocket, "inputs")
			}, "failed to connect to monitoring socket inputs endpoint, last error was: %s", &lastError)
			assert.Condition(c, func() bool {
				processorsLoaded := zapLogs.FilterMessageSnippet("Generated new processors").Len() > 0
				assert.False(c, processorsLoaded, "processors loaded but none expected")
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

func TestMultipleReceivers(t *testing.T) {
	monitorSocket1 := genSocketPath()
	var monitorHost1 string
	if runtime.GOOS == "windows" {
		monitorHost1 = "npipe:///" + filepath.Base(monitorSocket1)
	} else {
		monitorHost1 = "unix://" + monitorSocket1
	}
	monitorSocket2 := genSocketPath()
	var monitorHost2 string
	if runtime.GOOS == "windows" {
		monitorHost2 = "npipe:///" + filepath.Base(monitorSocket2)
	} else {
		monitorHost2 = "unix://" + monitorSocket2
	}
	config1 := Config{
		Beatconfig: map[string]any{
			"metricbeat": map[string]any{
				"modules": []map[string]any{
					{
						"module":     "system",
						"enabled":    true,
						"period":     "1s",
						"processes":  []string{".*"},
						"metricsets": []string{"cpu"},
					},
				},
			},
			"logging": map[string]any{
				"level": "debug",
				"selectors": []string{
					"*",
				},
			},
			"path.home":    t.TempDir(),
			"http.enabled": true,
			"http.host":    monitorHost1,
		},
	}

	config2 := Config{
		Beatconfig: map[string]any{
			"metricbeat": map[string]any{
				"modules": []map[string]any{
					{
						"module":     "system",
						"enabled":    true,
						"period":     "1s",
						"processes":  []string{".*"},
						"metricsets": []string{"cpu"},
					},
				},
			},
			"logging": map[string]any{
				"level": "debug",
				"selectors": []string{
					"*",
				},
			},
			"path.home":    t.TempDir(),
			"http.enabled": true,
			"http.host":    monitorHost2,
		},
	}

	factory := NewFactory()
	oteltest.CheckReceivers(oteltest.CheckReceiversParams{
		T: t,
		Receivers: []oteltest.ReceiverConfig{
			{
				Name:    "r1",
				Beat:    "metricbeat",
				Config:  &config1,
				Factory: factory,
			},
			{
				Name:    "r2",
				Beat:    "metricbeat",
				Config:  &config2,
				Factory: factory,
			},
		},
		AssertFunc: func(c *assert.CollectT, logs map[string][]mapstr.M, zapLogs *observer.ObservedLogs) {
			_ = zapLogs
			require.Conditionf(c, func() bool {
				return len(logs["r1"]) > 0 && len(logs["r2"]) > 0
			}, "expected at least one ingest log for each receiver, got logs: %v", logs)
			assert.Equal(c, "metricbeat", logs["r1"][0].Flatten()["agent.type"], "expected agent.type field to be 'metricbeat' in r1")
			assert.Equal(c, "metricbeat", logs["r2"][0].Flatten()["agent.type"], "expected agent.type field to be 'metricbeat' in r2")

			// Make sure that each receiver has a separate logger
			// instance and does not interfere with others. Previously, the
			// logger in Beats was global, causing logger fields to be
			// overwritten when multiple receivers started in the same process.
			r1StartLogs := zapLogs.FilterMessageSnippet("Beat ID").FilterField(zap.String("otelcol.component.id", "metricbeatreceiver/r1"))
			assert.Equal(c, 1, r1StartLogs.Len(), "r1 should have a single start log")
			r2StartLogs := zapLogs.FilterMessageSnippet("Beat ID").FilterField(zap.String("otelcol.component.id", "metricbeatreceiver/r2"))
			assert.Equal(c, 1, r2StartLogs.Len(), "r2 should have a single start log")

			r1StartMetricsLogs := zapLogs.FilterMessageSnippet("Starting metrics logging every 30s").FilterField(zap.String("otelcol.component.id", "metricbeatreceiver/r1"))
			assert.Equalf(c, 1, r1StartMetricsLogs.Len(), "r1 should have a single start metrircs logging every 30s")
			r2StartMetricsLogs := zapLogs.FilterMessageSnippet("Starting metrics logging every 30s").FilterField(zap.String("otelcol.component.id", "metricbeatreceiver/r1"))
			assert.Equalf(c, 1, r2StartMetricsLogs.Len(), "r2 should have a single start metrircs logging every 30s")

			var lastError strings.Builder
			assert.Conditionf(c, func() bool {
				tests := []string{monitorSocket1, monitorSocket2}
				for _, tc := range tests {
					if ret := getFromSocket(t, &lastError, tc, "stats"); ret == false {
						return false
					}
					if ret := getFromSocket(t, &lastError, tc, "inputs"); ret == false {
						return false
					}
				}
				return true
			}, "failed to connect to monitoring socket, last error was: %s", &lastError)
		},
	})
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

func BenchmarkFactory(b *testing.B) {
	tmpDir := b.TempDir()

	cfg := &Config{
		Beatconfig: map[string]interface{}{
			"metricbeat": map[string]any{
				"modules": []map[string]any{
					{
						"module":     "system",
						"enabled":    true,
						"period":     "1s",
						"processes":  []string{".*"},
						"metricsets": []string{"cpu"},
					},
				},
			},
			"logging": map[string]any{
				"level": "info",
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
		zapcore.InfoLevel)

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
	expectedDegradedErrorMessage := "Error fetching data for metricset benchmark.info: benchmark input degraded"
	testCases := []struct {
		name    string
		status  *componentstatus.Event
		degrade bool
	}{
		{
			name: "degraded input",
			status: componentstatus.NewEvent(
				componentstatus.StatusRecoverableError,
				componentstatus.WithError(errors.New(expectedDegradedErrorMessage)),
				componentstatus.WithAttributes(inputStatusAttributes(
					componentstatus.StatusRecoverableError.String(), expectedDegradedErrorMessage)),
			),
			degrade: true,
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
					"metricbeat": map[string]any{
						"max_start_delay": "0s",
						"modules": []map[string]any{
							{
								"id":         benchmarkInputId,
								"module":     "benchmark",
								"enabled":    true,
								"period":     "1s",
								"degrade":    test.degrade,
								"metricsets": []string{"info"},
							},
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
						Beat:    "metricbeat",
						Config:  &config,
						Factory: NewFactory(),
					},
				},
				Status: test.status,
			})
		})
	}
}

func TestReceiverHook(t *testing.T) {
	cfg := Config{
		Beatconfig: map[string]any{
			"metricbeat": map[string]any{
				"max_start_delay": "0s",
				"modules": []map[string]any{
					{
						"module":     "benchmark",
						"enabled":    true,
						"period":     "1s",
						"metricsets": []string{"info"},
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

	// For metricbeatreceiver, we expect 2 hooks to be registered:
	// 	one for beat metrics and one for input metrics.
	oteltest.TestReceiverHook(t, &cfg, NewFactory(), receiverSettings, 2)
}
