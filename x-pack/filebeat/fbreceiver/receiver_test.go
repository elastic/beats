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
	"strings"
	"testing"

	"github.com/elastic/beats/v7/libbeat/otelbeat/oteltest"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
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
			"output": map[string]any{
				"otelconsumer": map[string]any{},
			},
			"logging": map[string]any{
				"level": "debug",
				"selectors": []string{
					"*",
				},
			},
			"path.home":    t.TempDir(),
			"http.enabled": true,
			"http.host":    monitorHost,
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
					FilterMessageSnippet("add_kubernetes_metadata").
					Len() == 1
				assert.True(c, processorsLoaded, "processors not loaded")
				// Check that add_host_metadata works, other processors are not guaranteed to add fields in all environments
				return assert.Contains(c, logs["r1"][0].Flatten(), "host.architecture")
			}, "failed to check processors loaded")
		},
	})
}

func BenchmarkFactory(b *testing.B) {
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
			"output": map[string]any{
				"otelconsumer": map[string]any{},
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

func TestMultipleReceivers(t *testing.T) {
	// Receivers need distinct home directories so wrap the config in a function.
	config := func(monitorSocket string) *Config {
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
							"type":    "filestream",
							"enabled": true,
							"id":      "must-be-unique",
							"paths":   []string{"none"},
						},
					},
				},
				"output": map[string]any{
					"otelconsumer": map[string]any{},
				},
				"logging": map[string]any{
					"level": "info",
					"selectors": []string{
						"*",
					},
				},
				"path.home":    t.TempDir(),
				"http.enabled": true,
				"http.host":    monitorHost,
			},
		}
	}

	factory := NewFactory()
	monitorSocket1 := genSocketPath()
	monitorSocket2 := genSocketPath()
	oteltest.CheckReceivers(oteltest.CheckReceiversParams{
		T: t,
		Receivers: []oteltest.ReceiverConfig{
			{
				Name:    "r1",
				Config:  config(monitorSocket1),
				Beat:    "filebeat",
				Factory: factory,
			},
			{
				Name:    "r2",
				Config:  config(monitorSocket2),
				Beat:    "filebeat",
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
			fmt.Fprintf(sb, "json array didn't have any entries\nbody was: %v", body)
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
