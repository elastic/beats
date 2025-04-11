// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbreceiver

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

<<<<<<< HEAD
	var countLogs int
	logConsumer, err := consumer.NewLogs(func(ctx context.Context, ld plog.Logs) error {
		countLogs = countLogs + ld.LogRecordCount()
		return nil
=======
	factory := NewFactory()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := factory.CreateLogs(context.Background(), receiverSettings, cfg, nil)
		require.NoError(b, err)
	}
}

func TestMultipleReceivers(t *testing.T) {
	// This test verifies that multiple receivers can be instantiated
	// in isolation, started, and can ingest logs without interfering
	// with each other.
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

	factory := NewFactory()
	oteltest.CheckReceivers(oteltest.CheckReceiversParams{
		T: t,
		Receivers: []oteltest.ReceiverConfig{
			{
				Name:    "r1",
				Config:  &config,
				Factory: factory,
			},
			{
				Name:    "r2",
				Config:  &config,
				Factory: factory,
			},
		},
		AssertFunc: func(t *assert.CollectT, logs map[string][]mapstr.M, zapLogs *observer.ObservedLogs) {
			r1ok := assert.Greater(t, len(logs["r1"]), 0, "receive r1 does not have any logs")
			r2ok := assert.Greater(t, len(logs["r2"]), 0, "receive r2 does not have any logs")
			// logs for debug if it fails again
			fmt.Printf("len(logs[\"r1\"]): %d\n", len(logs["r1"]))
			fmt.Printf("len(logs[\"r2\"]): %d\n", len(logs["r2"]))
			if !r1ok || !r2ok {
				fmt.Printf("logs[\"r1\"]: %v\n", logs["r1"])
				fmt.Printf("logs[\"r2\"]: %v\n", logs["r2"])
				fmt.Printf("all logs: %v\n", logs)
			}

			// Make sure that each receiver has a separate logger
			// instance and does not interfere with others. Previously, the
			// logger in Beats was global, causing logger fields to be
			// overwritten when multiple receivers started in the same process.
			r1StartLogs := zapLogs.FilterMessageSnippet("Beat ID").FilterField(zap.String("name", "r1"))
			assert.Equal(t, 1, r1StartLogs.Len(), "r1 should have a single start log")
			r2StartLogs := zapLogs.FilterMessageSnippet("Beat ID").FilterField(zap.String("name", "r2"))
			require.Equal(t, 1, r2StartLogs.Len(), "r2 should have a single start log")
		},
>>>>>>> c9d6ad4c8 ([Flaky Test] oteltest.CheckReceivers increase timeout (#43834))
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
