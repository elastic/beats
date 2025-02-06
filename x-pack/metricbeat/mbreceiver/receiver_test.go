// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mbreceiver

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

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
