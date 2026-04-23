// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pbreceiver

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
	"time"

	"github.com/elastic/beats/v7/x-pack/otel/oteltest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
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
			"packetbeat": map[string]any{
				"interfaces": map[string]any{
					"device": "lo0",
				},
				"protocols": []map[string]any{
					{
						"type":  "http",
						"ports": []int{80, 8080},
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

	zapCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		&zaptest.Discarder{},
		zapcore.DebugLevel,
	)
	observed, zapLogs := observer.New(zapcore.DebugLevel)
	core := zapcore.NewTee(zapCore, observed)

	factory := NewFactoryWithSettings(Settings{Home: t.TempDir()})
	receiverSettings := receiver.Settings{}
	receiverSettings.ID = component.NewIDWithName(factory.Type(), "r1")
	receiverSettings.Logger = zap.New(core.With([]zapcore.Field{
		zap.String("otelcol.component.id", receiverSettings.ID.String()),
		zap.String("otelcol.component.kind", "receiver"),
		zap.String("otelcol.signal", "logs"),
	}))

	rec, err := factory.CreateLogs(t.Context(), receiverSettings, &config, consumertest.NewNop())
	require.NoError(t, err)
	require.NoError(t, rec.Start(t.Context(), componenttest.NewNopHost()))

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		// Verify the monitoring socket is available
		var lastError strings.Builder
		assert.Conditionf(c, func() bool {
			return getFromSocket(t, &lastError, monitorSocket, "stats")
		}, "failed to connect to monitoring socket stats endpoint, last error was: %s", &lastError)

		// Verify metrics logging started
		metricsStarted := zapLogs.FilterMessageSnippet("Starting metrics logging every 30s")
		assert.NotEmpty(c, metricsStarted.All(), "metrics logging not started")
	}, 2*time.Minute, 1*time.Second,
		"timeout waiting for packetbeat receiver to start")

	require.NoError(t, rec.Shutdown(t.Context()))
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

	// Each config needs a unique "id" so that packetbeat's per-protocol
	// input metrics (registered in a global monitoring registry) get
	// distinct names and don't collide.
	config1 := Config{
		Beatconfig: map[string]any{
			"packetbeat": map[string]any{
				"id": "receiver-1",
				"interfaces": map[string]any{
					"device": "lo0",
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
			"packetbeat": map[string]any{
				"id": "receiver-2",
				"interfaces": map[string]any{
					"device": "lo0",
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

	zapCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		&zaptest.Discarder{},
		zapcore.DebugLevel,
	)
	observed, zapLogs := observer.New(zapcore.DebugLevel)
	core := zapcore.NewTee(zapCore, observed)

	factory := NewFactoryWithSettings(Settings{Home: t.TempDir()})

	createReceiver := func(name string, cfg *Config) receiver.Logs {
		set := receiver.Settings{}
		set.ID = component.NewIDWithName(factory.Type(), name)
		set.Logger = zap.New(core.With([]zapcore.Field{
			zap.String("otelcol.component.id", set.ID.String()),
			zap.String("otelcol.component.kind", "receiver"),
			zap.String("otelcol.signal", "logs"),
		}))
		r, err := factory.CreateLogs(t.Context(), set, cfg, consumertest.NewNop())
		require.NoError(t, err)
		return r
	}

	r1 := createReceiver("r1", &config1)
	r2 := createReceiver("r2", &config2)

	require.NoError(t, r1.Start(t.Context(), componenttest.NewNopHost()))
	require.NoError(t, r2.Start(t.Context(), componenttest.NewNopHost()))

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		// Make sure that each receiver has a separate logger instance
		r1StartLogs := zapLogs.FilterMessageSnippet("Beat ID").FilterField(zap.String("otelcol.component.id", "packetbeatreceiver/r1"))
		assert.Equal(c, 1, r1StartLogs.Len(), "r1 should have a single start log")
		r2StartLogs := zapLogs.FilterMessageSnippet("Beat ID").FilterField(zap.String("otelcol.component.id", "packetbeatreceiver/r2"))
		assert.Equal(c, 1, r2StartLogs.Len(), "r2 should have a single start log")

		r1StartMetricsLogs := zapLogs.FilterMessageSnippet("Starting metrics logging every 30s").FilterField(zap.String("otelcol.component.id", "packetbeatreceiver/r1"))
		assert.Equalf(c, 1, r1StartMetricsLogs.Len(), "r1 should have a single start metrics logging every 30s")
		r2StartMetricsLogs := zapLogs.FilterMessageSnippet("Starting metrics logging every 30s").FilterField(zap.String("otelcol.component.id", "packetbeatreceiver/r2"))
		assert.Equalf(c, 1, r2StartMetricsLogs.Len(), "r2 should have a single start metrics logging every 30s")

		// Verify monitoring sockets
		var lastError strings.Builder
		assert.Conditionf(c, func() bool {
			for _, sock := range []string{monitorSocket1, monitorSocket2} {
				if !getFromSocket(t, &lastError, sock, "stats") {
					return false
				}
			}
			return true
		}, "failed to connect to monitoring socket, last error was: %s", &lastError)
	}, 2*time.Minute, 1*time.Second,
		"timeout waiting for packetbeat receivers to start")

	require.NoError(t, r1.Shutdown(t.Context()))
	require.NoError(t, r2.Shutdown(t.Context()))
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
	return true
}

func BenchmarkFactory(b *testing.B) {
	tmpDir := b.TempDir()

	cfg := &Config{
		Beatconfig: map[string]interface{}{
			"packetbeat": map[string]any{
				"interfaces": map[string]any{
					"device": "lo0",
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

	factory := NewFactoryWithSettings(Settings{Home: tmpDir})

	receiverSettings := receiver.Settings{}
	receiverSettings.Logger = zap.New(core)
	receiverSettings.ID = component.NewIDWithName(factory.Type(), "r1")

	b.ResetTimer()
	for b.Loop() {
		_, err := factory.CreateLogs(b.Context(), receiverSettings, cfg, nil)
		require.NoError(b, err)
	}
}

func TestReceiverHook(t *testing.T) {
	cfg := Config{
		Beatconfig: map[string]any{
			"packetbeat": map[string]any{
				"interfaces": map[string]any{
					"device": "lo0",
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

	// For packetbeatreceiver, we expect 2 hooks to be registered:
	// 	one for beat metrics and one for input metrics.
	oteltest.TestReceiverHook(t, &cfg, NewFactoryWithSettings(Settings{Home: t.TempDir()}), receiverSettings, 2)
}
