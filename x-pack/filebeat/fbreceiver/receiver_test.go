// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbreceiver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
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
	tmpDir := t.TempDir()

	monitorSocket, monitorHost := genSocketPath()
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
			"path.home":    tmpDir,
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
		AssertFunc: func(ct *assert.CollectT, logs map[string][]mapstr.M, zapLogs *observer.ObservedLogs) {
			require.Lenf(ct, logs["r1"], 1, "expected 1 log, got %d", len(logs["r1"]))

			assertGetFromSocket(t, monitorSocket, false)

			assert.Condition(ct, func() bool {
				processorsLoaded := zapLogs.FilterMessageSnippet("Generated new processors").
					FilterMessageSnippet("add_host_metadata").
					FilterMessageSnippet("add_cloud_metadata").
					FilterMessageSnippet("add_docker_metadata").
					FilterMessageSnippet("add_kubernetes_metadata").
					Len() == 1
				assert.True(ct, processorsLoaded, "processors not loaded")
				// Check that add_host_metadata works, other processors are not guaranteed to add fields in all environments
				return assert.Contains(ct, logs["r1"][0].Flatten(), "host.architecture")
			}, "failed to check processors loaded")
		},
	})
}

func TestFactory(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := t.Context()

	monitorSocket, monitorHost := genSocketPath()
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
				"level": "info",
				"selectors": []string{
					"*",
				},
			},
			"path.home":    tmpDir,
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
	assertGetFromSocket(t, monitorSocket, true)
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

func assertGetFromSocket(t *testing.T, socketPath string, skipBodyCheck bool) {
	t.Helper()
	// skip windows for now
	if runtime.GOOS == "windows" {
		return
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
		require.NoErrorf(t, err, "error creating request to %q: %s", endpoint, err)
		resp, err := client.Do(req)
		require.NoErrorf(t, err, "client.Get failed for %q: %s", endpoint, err)
		defer resp.Body.Close()

		require.Equal(t, resp.StatusCode, http.StatusOK, "unexpected status code for %q: %d", endpoint, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoErrorf(t, err, "io.ReadAll of body failed for %q: %s", endpoint, err)
		t.Logf("metrics endpoint %q body: %s", endpoint, string(body))
		if skipBodyCheck {
			return
		}

		require.GreaterOrEqualf(t, len(body), 5, "body too short for %q: %s", endpoint, body)

		switch endpoint {
		case "inputs/":
			var bodyMap []mapstr.M
			err := json.Unmarshal(body, &bodyMap)
			require.NoErrorf(t, err, "json.Unmarshal failed for %q: %s", endpoint, err)
			require.Greaterf(t, len(bodyMap), 0, "body is empty for %q", endpoint)
			for _, v := range bodyMap {
				require.Containsf(t, v, "input", "body does not contain input key for %q", endpoint)
				require.Equalf(t, v["input"], "benchmark", "expected input type benchmark for %q, got %s", endpoint, v["input"])
			}
		case "stats/":
			var bodyMap mapstr.M
			err := json.Unmarshal(body, &bodyMap)
			require.NoErrorf(t, err, "json.Unmarshal failed for %q: %s", endpoint, err)
			require.Greaterf(t, len(bodyMap), 0, "body is empty for %q", endpoint)
			require.Containsf(t, bodyMap, "beat", "body does not contain beat key for %q", endpoint)
		default:
		}
	}
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

func TestMultipleReceivers(t *testing.T) {
	// This test verifies that multiple receivers can be instantiated
	// in isolation, started, and can ingest logs without interfering
	// with each other.

	monitorSocket1, monitorHost1 := genSocketPath()
	monitorSocket2, monitorHost2 := genSocketPath()
	// Receivers need distinct home directories so wrap the config in a function.
	config1 := Config{
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
			"path.home":    t.TempDir(),
			"http.enabled": true,
			"http.host":    monitorHost1,
		},
	}

	config2 := Config{
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
			require.Greater(c, len(logs["r1"]), 0, "receiver r1 does not have any logs")
			require.Greater(c, len(logs["r2"]), 0, "receiver r2 does not have any logs")

			tests := []string{monitorSocket1, monitorSocket2}
			for _, tc := range tests {
				assertGetFromSocket(t, tc, false)
			}

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
