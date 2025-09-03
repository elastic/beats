// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mbreceiver

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/goleak"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLeak(t *testing.T) {
	// goroutine comes from init in cloud.google.com/go/pubsub and filebeat/input/gcppubsub
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"))

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
			"output": map[string]any{
				"otelconsumer": map[string]any{},
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
			"queue.mem.flush.timeout": "0s",
		},
	}
	factory := NewFactory()
	var receiverSettings receiver.Settings
	observedCore, observedLogs := observer.New(zapcore.DebugLevel)
	receiverSettings.Logger = zap.New(observedCore)
	receiverSettings.ID = component.NewIDWithName(factory.Type(), "r1")

	var consumeLogs DummyConsumer
	rec, err := factory.CreateLogs(t.Context(), receiverSettings, &config, &consumeLogs)
	require.NoError(t, err)
	require.NoError(t, rec.Start(t.Context(), nil))
	if !assert.Eventually(t,
		func() bool {
			return observedLogs.FilterMessageSnippet("system/cpu will start after").Len() >= 1
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

type DummyConsumer struct {
	context.Context
}

func (d *DummyConsumer) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	return nil
}

func (d *DummyConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{}
}
