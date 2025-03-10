// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbreceiver

import (
	"bufio"
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/otelbeat/oteltest"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNewReceiver(t *testing.T) {
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
			"path.home": t.TempDir(),
		},
	}

	var zapLogs bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&zapLogs),
		zapcore.DebugLevel)

	receiverSettings := receiver.Settings{}
	receiverSettings.Logger = zap.New(core)

	var countLogs int
	logConsumer, err := consumer.NewLogs(func(ctx context.Context, ld plog.Logs) error {
		countLogs = countLogs + ld.LogRecordCount()
		return nil
	})
	assert.NoError(t, err, "Error creating log consumer")

	r, err := createReceiver(context.Background(), receiverSettings, &config, logConsumer)
	assert.NoErrorf(t, err, "Error creating receiver. Logs:\n %s", zapLogs.String())
	err = r.Start(context.Background(), nil)
	assert.NoError(t, err, "Error starting filebeatreceiver")

	ch := make(chan bool, 1)
	timer := time.NewTimer(120 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for tick := ticker.C; ; {
		select {
		case <-timer.C:
			t.Fatalf("consumed logs didn't increase\nCount: %d\nLogs: %s\n", countLogs, zapLogs.String())
		case <-tick:
			tick = nil
			go func() { ch <- countLogs > 0 }()
		case v := <-ch:
			if v {
				goto found
			}
			tick = ticker.C
		}
	}
found:
	err = r.Shutdown(context.Background())
	assert.NoError(t, err, "Error shutting down filebeatreceiver")
}

func TestReceiverDefaultProcessors(t *testing.T) {
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
			"path.home": t.TempDir(),
		},
	}

	var zapLogs bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&zapLogs),
		zapcore.DebugLevel)

	receiverSettings := receiver.Settings{}
	receiverSettings.Logger = zap.New(core)

	logsCh := make(chan []mapstr.M, 1)
	logConsumer, err := consumer.NewLogs(func(ctx context.Context, ld plog.Logs) error {
		var logs []mapstr.M
		for i := 0; i < ld.ResourceLogs().Len(); i++ {
			rl := ld.ResourceLogs().At(i)
			for j := 0; j < rl.ScopeLogs().Len(); j++ {
				sl := rl.ScopeLogs().At(j)
				for k := 0; k < sl.LogRecords().Len(); k++ {
					log := sl.LogRecords().At(k)
					logs = append(logs, log.Body().Map().AsRaw())
				}
			}
		}

		logsCh <- logs
		return nil
	})
	assert.NoError(t, err, "Error creating log consumer")

	r, err := NewFactory().CreateLogs(context.Background(), receiverSettings, &config, logConsumer)
	assert.NoErrorf(t, err, "Error creating receiver. Logs:\n %s", zapLogs.String())

	err = r.Start(context.Background(), nil)
	assert.NoError(t, err, "Error starting filebeatreceiver")
	defer func() {
		require.NoError(t, r.Shutdown(context.Background()))
	}()

	var logs []mapstr.M
	select {
	case logs = <-logsCh:
	case <-time.After(1 * time.Minute):
		t.Fatal("timeout waiting for logs")
	}

	require.Len(t, logs, 1)
	t.Log("ingested log: ", logs[0])

	scanner := bufio.NewScanner(&zapLogs)
	wantKeywords := []string{
		"Generated new processors",
		"add_host_metadata",
		"add_cloud_metadata",
		"add_docker_metadata",
		"add_kubernetes_metadata",
	}

	var processorsLoaded bool
	for scanner.Scan() {
		line := scanner.Text()
		if stringContainsAll(line, wantKeywords) {
			processorsLoaded = true
			break
		}
	}

	require.True(t, processorsLoaded, "processors not loaded")
	// Check that add_host_metadata works, other processors are not guaranteed to add fields in all environments
	require.Contains(t, logs[0].Flatten(), "host.architecture")
}

func stringContainsAll(s string, want []string) bool {
	for _, w := range want {
		if !strings.Contains(s, w) {
			return false
		}
	}

	return true
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
				"level": "debug",
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
		zapcore.AddSync(&zapLogs),
		zapcore.DebugLevel)

	receiverSettings := receiver.Settings{}
	receiverSettings.Logger = zap.New(core)

	factory := NewFactory()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := factory.CreateLogs(context.Background(), receiverSettings, cfg, nil)
		require.NoError(b, err)
	}
}

func TestMultipleReceivers(t *testing.T) {
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
			"path.home": t.TempDir(),
		},
	}

	oteltest.CheckMultipleReceivers(oteltest.CheckMultipleReceiversParams{
		T:       t,
		Factory: NewFactory(),
		Config:  &config,
		AssertFunc: func(t *testing.T, logs map[string][]mapstr.M) {
			require.Eventuallyf(t, func() bool {
				return len(logs["r1"]) == 1 && len(logs["r2"]) == 1
			}, 1*time.Minute, 100*time.Millisecond, "timeout waiting for logs: %#v", logs)
		},
	})
}
