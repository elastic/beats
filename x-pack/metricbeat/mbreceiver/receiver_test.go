// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mbreceiver

import (
	"bytes"
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewReceiver(t *testing.T) {
	config := Config{
		Beatconfig: map[string]interface{}{
			"metricbeat": map[string]interface{}{
				"modules": []map[string]interface{}{
					{
						"module":     "system",
						"enabled":    true,
						"period":     "1s",
						"processes":  []string{".*"},
						"metricsets": []string{"cpu"},
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
			"path.home": t.TempDir(),
		},
	}

	core, logs := observer.New(zapcore.DebugLevel)

	receiverSettings := receiver.Settings{}
	receiverSettings.Logger = zap.New(core)

	var countLogs atomic.Int64
	logConsumer, err := consumer.NewLogs(func(ctx context.Context, ld plog.Logs) error {
		countLogs.Add(int64(ld.LogRecordCount()))
		return nil
	})
	require.NoError(t, err, "Error creating log consumer")

	r, err := createReceiver(context.Background(), receiverSettings, &config, logConsumer)
	require.NoErrorf(t, err, "Error creating receiver. Logs:\n %s", logs.All())
	err = r.Start(context.Background(), nil)
	require.NoError(t, err, "Error starting metricbeatreceiver")

	ch := make(chan bool, 1)
	timer := time.NewTimer(120 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for tick := ticker.C; ; {
		select {
		case <-timer.C:
			t.Fatalf("consumed logs didn't increase\nCount: %d\nLogs: %v\n", countLogs.Load(), logs.All())
		case <-tick:
			tick = nil
			go func() { ch <- countLogs.Load() > 0 }()
		case v := <-ch:
			if v {
				goto found
			}
			tick = ticker.C
		}
	}
found:
	err = r.Shutdown(context.Background())
	require.NoError(t, err, "Error shutting down metricbeatreceiver")
}

func TestMultipleReceivers(t *testing.T) {
	logs := make(map[string]int)

	ctx := context.Background()
	createReceiver := func(t *testing.T, name string) receiver.Logs {
		t.Helper()
		config := Config{
			Beatconfig: map[string]interface{}{
				"metricbeat": map[string]interface{}{
					"modules": []map[string]interface{}{
						{
							"module":     "system",
							"enabled":    true,
							"period":     "1s",
							"processes":  []string{".*"},
							"metricsets": []string{"cpu"},
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
				"path.home": t.TempDir(),
			},
		}

		var zapLogs bytes.Buffer
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(&zapLogs),
			zapcore.DebugLevel)

		receiverSettings := receiver.Settings{}
		receiverSettings.Logger = zap.New(core).Named(name)

		logConsumer, err := consumer.NewLogs(func(ctx context.Context, ld plog.Logs) error {
			for i := 0; i < ld.ResourceLogs().Len(); i++ {
				rl := ld.ResourceLogs().At(i)
				for j := 0; j < rl.ScopeLogs().Len(); j++ {
					sl := rl.ScopeLogs().At(j)
					for k := 0; k < sl.LogRecords().Len(); k++ {
						log := sl.LogRecords().At(k)
						logs[name] = logs[name] + 1
						t.Logf("ingested log for %q: %v", name, log.Body().Map().AsRaw())
					}
				}
			}
			return nil
		})
		assert.NoErrorf(t, err, "Error creating log consumer for %q", name)

		t.Cleanup(func() {
			if t.Failed() {
				t.Logf("Logs for %q: %s\n", name, zapLogs.String())
			}
		})

		r, err := NewFactory().CreateLogs(ctx, receiverSettings, &config, logConsumer)
		assert.NoErrorf(t, err, "Error creating receiver %q", name)
		return r
	}

	r1 := createReceiver(t, "r1")
	r2 := createReceiver(t, "r2")

	err := r1.Start(ctx, nil)
	require.NoError(t, err, "Error starting receiver 1")
	defer func() {
		require.NoError(t, r1.Shutdown(ctx))
	}()

	err = r2.Start(ctx, nil)
	require.NoError(t, err, "Error starting receiver 2")
	defer func() {
		require.NoError(t, r2.Shutdown(ctx))
	}()

	require.Eventuallyf(t, func() bool {
		return logs["r1"] > 1 && logs["r2"] > 1
	}, 1*time.Minute, 100*time.Millisecond, "timeout waiting for logs: %#v", logs)
}
