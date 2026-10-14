// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqreceiver

import (
	"bytes"
	"context"
	"encoding/base64"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
)

// NOTE: TestNewReceiver, TestMultipleReceivers, and the "running input"
// sub-test of TestReceiverStatus are not included here because they require a
// live osqueryd binary (see tests/integration/receiver_test.go).
//
// TestReceiverStatus verifies that a missing daemon is reported as a terminal
// input failure without announcing readiness before the runtime check.

func BenchmarkFactory(b *testing.B) {
	tmpDir := b.TempDir()

	cfg := &Config{
		Beatconfig: map[string]any{
			"osquerybeat": map[string]any{
				"inputs": []any{
					map[string]any{
						"type": "osquery",
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

	factory := NewFactoryWithSettings(Settings{Home: tmpDir})

	receiverSettings := receiver.Settings{}
	receiverSettings.Logger = zap.New(core)
	receiverSettings.ID = component.NewIDWithName(factory.Type(), "r1")

	b.ResetTimer()
	for b.Loop() {
		rcvr, err := factory.CreateLogs(b.Context(), receiverSettings, cfg, nil)
		require.NoError(b, err)
		err = rcvr.Shutdown(b.Context())
		require.NoError(b, err)
	}
}

func TestReceiverHook(t *testing.T) {
	cfg := Config{
		Beatconfig: map[string]any{
			"osquerybeat": map[string]any{
				"inputs": []any{
					map[string]any{
						"type": "osquery",
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

	// For osquerybeatreceiver, we expect 3 hooks: one for beat metrics, one for
	// scheduled query profiles, and one for customer-managed extensions,
	// all registered by osquerybeat.registerDiagnosticHooks.
	oteltest.TestReceiverHook(t, &cfg, NewFactoryWithSettings(Settings{Home: t.TempDir()}), receiverSettings, 3)
}

func TestReceiverStatus(t *testing.T) {
	const inputID = "osquery-status-test"
	// Select an empty binary directory even when the host has osquery installed.
	t.Setenv("OSQUERYBEAT_BINARY_DIR", t.TempDir())
	t.Cleanup(func() { oteltest.VerifyNoLeaks(t) })
	cfg := &Config{Beatconfig: map[string]any{
		"osquerybeat": map[string]any{"inputs": []any{
			map[string]any{"type": "osquery", "id": inputID},
		}},
		"management.otel.enabled": true,
		"http.enabled":            false,
		"path.home":               t.TempDir(),
	}}
	factory := NewFactoryWithSettings(Settings{Home: t.TempDir()})
	settings := receiver.Settings{
		ID:                component.NewIDWithName(factory.Type(), "status"),
		TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
	}
	host := &oteltest.MockHost{}
	var consumeLogs oteltest.DummyConsumer
	rec, err := factory.CreateLogs(t.Context(), settings, cfg, &consumeLogs)
	require.NoError(t, err, "create receiver with missing daemon")
	require.NoError(t, rec.Start(t.Context(), host), "start receiver lifecycle")
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		require.NoError(t, rec.Shutdown(ctx), "clean up receiver after runtime failure")
	})
	require.Eventually(t, func() bool {
		for _, event := range host.GetEvents() {
			if event.Status() != componentstatus.StatusPermanentError {
				continue
			}
			inputs, ok := event.Attributes().Get("inputs")
			if !ok {
				continue
			}
			input, ok := inputs.Map().Get(inputID)
			if !ok {
				continue
			}
			state, ok := input.Map().Get("status")
			if ok && state.Str() == componentstatus.StatusPermanentError.String() {
				return true
			}
		}
		return false
	}, 10*time.Second, 10*time.Millisecond, "missing daemon must fail its identified input")
	for _, event := range host.GetEvents() {
		require.NotEqual(t, componentstatus.StatusOK, event.Status(), "missing daemon must never announce readiness")
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
