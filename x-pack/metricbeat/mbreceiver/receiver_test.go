// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mbreceiver

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

func TestFactory(t *testing.T) {
	ctx := t.Context()
	monitorSocket, monitorHost := genSocketPath()
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

	var zapLogs bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.Lock(zapcore.AddSync(&zapLogs)),
		zapcore.InfoLevel)

	factory := NewFactory()

	receiverSettings := receiver.Settings{}
	receiverSettings.Logger = zap.New(core)
	receiverSettings.ID = component.NewIDWithName(factory.Type(), "r1")

	rc, err := factory.CreateLogs(ctx, receiverSettings, cfg, nil)
	require.NotEmpty(t, rc, "receiver should not be empty")
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, rc.Shutdown(ctx))
	})

	// Ensure http metrics endpoint is reachable on receiver creation
	var lastError strings.Builder
	assert.Conditionf(t, func() bool {
		return getFromSocket(t, &lastError, monitorSocket, true)
	}, "failed to connect to monitoring socket, last error was: %s", &lastError)
}

func TestNewReceiver(t *testing.T) {
	monitorSocket, monitorHost := genSocketPath()
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
				Config:  &config,
				Factory: NewFactory(),
			},
		},
		AssertFunc: func(c *assert.CollectT, logs map[string][]mapstr.M, zapLogs *observer.ObservedLogs) {
			_ = zapLogs
			require.Conditionf(c, func() bool {
				return len(logs["r1"]) > 0
			}, "expected at least one ingest log, got logs: %v", logs["r1"])
			var lastError strings.Builder
			assert.Conditionf(c, func() bool {
				return getFromSocket(t, &lastError, monitorSocket, false)
			}, "failed to connect to monitoring socket, last error was: %s", &lastError)
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

func TestMultipleReceivers(t *testing.T) {
	monitorSocket1, monitorHost1 := genSocketPath()
	monitorSocket2, monitorHost2 := genSocketPath()
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
			"http.host":    monitorHost2,
		},
	}

	factory := NewFactory()
	oteltest.CheckReceivers(oteltest.CheckReceiversParams{
		T: t,
		Receivers: []oteltest.ReceiverConfig{
			{
				Name:    "r1",
				Config:  &config1,
				Factory: factory,
			},
			{
				Name:    "r2",
				Config:  &config2,
				Factory: factory,
			},
		},
		AssertFunc: func(c *assert.CollectT, logs map[string][]mapstr.M, zapLogs *observer.ObservedLogs) {
			_ = zapLogs
			require.Conditionf(c, func() bool {
				return len(logs["r1"]) > 0 && len(logs["r2"]) > 0
			}, "expected at least one ingest log for each receiver, got logs: %v", logs)
			var lastError strings.Builder
			assert.Conditionf(c, func() bool {
				tests := []string{monitorSocket1, monitorSocket2}
				for _, tc := range tests {
					if ret := getFromSocket(t, &lastError, tc, false); ret == false {
						return false
					}
				}
				return true
			}, "failed to connect to monitoring socket, last error was: %s", &lastError)

			assert.Condition(c, func() bool {
				processorsLoaded := zapLogs.FilterMessageSnippet("Generated new processors").
					FilterMessageSnippet("add_host_metadata").
					FilterMessageSnippet("add_cloud_metadata").
					FilterMessageSnippet("add_docker_metadata").
					FilterMessageSnippet("add_kubernetes_metadata").
					Len() == 2
				assert.True(c, processorsLoaded, "processors not loaded")
				// Check that add_host_metadata works, other processors are not guaranteed to add fields in all environments
				assert.Contains(c, logs["r1"][0].Flatten(), "host.architecture")
				return assert.Contains(c, logs["r2"][0].Flatten(), "host.architecture")
			}, "failed to check processors loaded")
		},
	})
}

func genSocketPath() (socketPath string, socketHost string) {
	randData := make([]byte, 16)
	for i := range len(randData) {
		randData[i] = uint8(rand.UintN(255)) //nolint:gosec // 0-255 fits in a uint8
	}
	socketName := base64.URLEncoding.EncodeToString(randData) + ".sock"
	socketDir := os.TempDir()
	socketPath = filepath.Join(socketDir, socketName)

	switch runtime.GOOS {
	case "windows":
		socketHost = "npipe:///" + filepath.Base(socketPath)
	default:
		socketHost = "unix://" + socketPath
	}

	return
}

func getFromSocket(t *testing.T, sb *strings.Builder, socketPath string, allowEmpty bool) bool {
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

	for _, endpoint := range []string{"inputs/", "stats/"} {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://unix/"+endpoint, nil)
		if err != nil {
			sb.Reset()
			fmt.Fprintf(sb, "%s: error creating request: %s", endpoint, err)
			return false
		}
		resp, err := client.Do(req)
		if err != nil {
			sb.Reset()
			fmt.Fprintf(sb, "%s: client.Get failed: %s", endpoint, err)
			return false
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			sb.Reset()
			fmt.Fprintf(sb, "%s: unexpected status code: %d", endpoint, resp.StatusCode)
			return false
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			sb.Reset()
			fmt.Fprintf(sb, "%s: io.ReadAll of body failed: %s", endpoint, err)
			return false
		}
		t.Logf("metrics endpoint %q body: %s", endpoint, string(body))
		if allowEmpty {
			return true
		}

		if len(body) <= 5 {
			sb.Reset()
			fmt.Fprintf(sb, "%s: body too short: %s", endpoint, body)
			return false
		}

		switch endpoint {
		case "inputs/":
			var bodyMap []mapstr.M
			if err := json.Unmarshal(body, &bodyMap); err != nil {
				sb.Reset()
				fmt.Fprintf(sb, "%s: json.Unmarshal failed: %s", endpoint, err)
				return false
			}

			if len(bodyMap) == 0 {
				sb.Reset()
				fmt.Fprintf(sb, "%s: body is empty", endpoint)
				return false
			}
			for _, v := range bodyMap {
				if _, ok := v["input"]; !ok {
					sb.Reset()
					fmt.Fprintf(sb, "%s: body does not contain input key", endpoint)
					return false
				}

				if v["input"] != "system/cpu" {
					sb.Reset()
					fmt.Fprintf(sb, "%s: unexpected input type: %s", endpoint, v["input"])
					return false
				}
			}
		case "stats/":
			var bodyMap mapstr.M
			if err := json.Unmarshal(body, &bodyMap); err != nil {
				sb.Reset()
				fmt.Fprintf(sb, "%s: json.Unmarshal failed: %s", endpoint, err)
				return false
			}
			if len(bodyMap) == 0 {
				sb.Reset()
				fmt.Fprintf(sb, "%s: body is empty", endpoint)
				return false
			}
			if _, ok := bodyMap["beat"]; !ok {
				sb.Reset()
				fmt.Fprintf(sb, "%s: body does not contain beat key", endpoint)
				return false
			}
		default:
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
	for i := 0; i < b.N; i++ {
		_, err := factory.CreateLogs(b.Context(), receiverSettings, cfg, nil)
		require.NoError(b, err)
	}
}
