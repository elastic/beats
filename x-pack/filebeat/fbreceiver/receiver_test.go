// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbreceiver

import (
	"bytes"
	"context"
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
			},
			"path.home": t.TempDir(),
		},
	}
	var zapLogs bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&zapLogs),
		zapcore.InfoLevel)

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

	assert.Eventuallyf(t, func() bool { return countLogs > 0 }, 60*time.Second, 1*time.Second, "consumed logs didn't increase\nCount: %d\nLogs: %s\n", countLogs, zapLogs.String())

	err = r.Shutdown(context.Background())
	assert.NoError(t, err, "Error shutting down filebeatreceiver")
}
