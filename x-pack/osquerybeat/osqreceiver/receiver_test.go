// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqreceiver

import (
	"bytes"
	"encoding/base64"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
)

// NOTE: TestNewReceiver and TestMultipleReceivers are not included here because
// osquerybeat requires osqueryd to be installed and running. Unlike metricbeat
// or auditbeat, osquerybeat cannot produce data without the external osqueryd
// binary. The existing tests (config, leak, hook, benchmark) cover the receiver
// wiring; beat-level integration testing is handled by the osquerybeat test
// suite which has mocking infrastructure for osqueryd.
//
// TestReceiverStatus IS included here as a unit test because the status.Running
// event is emitted before osqueryd setup in Run() — it fires as soon as the
// otelStatusFactoryWrapper creates the per-input runner and calls Start() on it.
// No osqueryd binary is required for that signal to reach the OTel host.

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

	// For osquerybeatreceiver, we expect 2 hooks: one for beat metrics and one for
	// scheduled query profiles registered by osquerybeat.registerDiagnosticHooks.
	oteltest.TestReceiverHook(t, &cfg, NewFactoryWithSettings(Settings{Home: t.TempDir()}), receiverSettings, 2)
}

func TestReceiverStatus(t *testing.T) {
	// inputID is embedded in the input config and picked up by getInputId so
	// the status attributes can reference a known, stable key.
	const inputID = "osquery-status-test"

	inputStatusAttributes := func(state, msg string) pcommon.Map {
		attrs := pcommon.NewMap()
		inputs := attrs.PutEmptyMap("inputs")
		inp := inputs.PutEmptyMap(inputID)
		inp.PutStr("status", state)
		inp.PutStr("error", msg)
		return attrs
	}

	t.Run("running input", func(t *testing.T) {
		cfg := Config{
			Beatconfig: map[string]any{
				"osquerybeat": map[string]any{
					"inputs": []any{
						map[string]any{
							"type": "osquery",
							"id":   inputID,
							"osquery": map[string]any{
								"schedule": map[string]any{
									"osquery_info": map[string]any{
										"query":    "SELECT * FROM osquery_info",
										"interval": 60,
									},
								},
							},
						},
					},
				},
				"path.home": t.TempDir(),
			},
		}
		// otelStatusFactoryWrapper fires runner.Start() before osqueryd setup, so
		// StatusOK reaches the host without requiring the osqueryd binary.
		oteltest.CheckReceivers(oteltest.CheckReceiversParams{
			T: t,
			Receivers: []oteltest.ReceiverConfig{
				{
					Name:    "r1",
					Beat:    "osquerybeat",
					Config:  &cfg,
					Factory: NewFactoryWithSettings(Settings{Home: t.TempDir()}),
				},
			},
			Status: componentstatus.NewEvent(
				componentstatus.StatusOK,
				componentstatus.WithAttributes(
					inputStatusAttributes(componentstatus.StatusOK.String(), ""))),
		})
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
