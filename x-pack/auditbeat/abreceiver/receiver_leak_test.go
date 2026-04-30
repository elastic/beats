// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package abreceiver

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
)

func TestLeak(t *testing.T) {
	t.Skip("See https://github.com/elastic/beats/issues/50381")
	monitorSocket := genSocketPath()
	var monitorHost string
	if runtime.GOOS == "windows" {
		monitorHost = "npipe:///" + filepath.Base(monitorSocket)
	} else {
		monitorHost = "unix://" + monitorSocket
	}
	config := Config{
		Beatconfig: map[string]any{
			"auditbeat": map[string]any{
				"modules": []map[string]any{
					{
						"module":        "file_integrity",
						"enabled":       true,
						"paths":         []string{t.TempDir()},
						"scan_at_start": false,
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
			"http.host":    monitorHost,
		},
	}
	factory := NewFactoryWithSettings(Settings{Home: t.TempDir()})

	t.Run("healthy consumer", func(t *testing.T) {
		defer oteltest.VerifyNoLeaks(t)
		var consumeLogs oteltest.DummyConsumer
		startAndStopReceiver(t, factory, &consumeLogs, &config)
	})
	t.Run("unhealthy consumer", func(t *testing.T) {
		defer oteltest.VerifyNoLeaks(t)
		consumeLogs := oteltest.DummyConsumer{ConsumeError: errors.New("cannot publish data")}
		startAndStopReceiver(t, factory, &consumeLogs, &config)
	})
}

// startAndStopReceiver creates a receiver using the provided parameters, starts it, verifies that the expected logs
// are output, then shuts it down, and verifies the logs again.
func startAndStopReceiver(t *testing.T, factory receiver.Factory, consumer consumer.Logs, config component.Config) {
	t.Helper()
	var receiverSettings receiver.Settings
	observedCore, observedLogs := observer.New(zapcore.DebugLevel)
	receiverSettings.Logger = zap.New(observedCore)
	receiverSettings.ID = component.NewIDWithName(factory.Type(), "r1")

	rec, err := factory.CreateLogs(t.Context(), receiverSettings, config, consumer)
	require.NoError(t, err)
	require.NoError(t, rec.Start(t.Context(), componenttest.NewNopHost()))
	if !assert.Eventually(t,
		func() bool {
			return observedLogs.FilterMessageSnippet("file_integrity/file will start after").Len() >= 1
		},
		60*time.Second,
		1*time.Second,
		"receiver not started") {
		for _, logLine := range observedLogs.TakeAll() {
			t.Log(logLine)
		}
		t.Fatalf("receiver didn't start, see logs above")
	}
	require.NoError(t, rec.Shutdown(t.Context()))
	if !assert.Eventually(t,
		func() bool {
			return observedLogs.FilterMessageSnippet("http: Server closed").Len() >= 1
		},
		60*time.Second,
		1*time.Second,
		"receiver not stopped") {
		for _, logLine := range observedLogs.TakeAll() {
			t.Log(logLine)
		}
		t.Fatalf("receiver didn't stop, see logs above")
	}
}
