// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Package oteltest provides test utilities for OpenTelemetry and Beats components.
package oteltest

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type CheckMultipleReceiversParams struct {
	T *testing.T
	// Factory that allows to create a receiver.
	Factory receiver.Factory
	// Config of the receiver to use.
	Config component.Config
	// AssertFunc is a function that asserts the test conditions.
	AssertFunc func(t *testing.T, logs map[string]int)
}

// CheckMultipleReceivers checks that multiple receivers can be created and started
// on the same process without errors.
func CheckMultipleReceivers(params CheckMultipleReceiversParams) {
	t := params.T
	logs := make(map[string]int)

	ctx := context.Background()
	createReceiver := func(t *testing.T, name string) receiver.Logs {
		t.Helper()

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

		r, err := params.Factory.CreateLogs(ctx, receiverSettings, params.Config, logConsumer)
		assert.NoErrorf(t, err, "Error creating receiver %q", name)
		return r
	}

	r1 := createReceiver(t, "r1")
	r2 := createReceiver(t, "r2")

	err := r1.Start(ctx, nil)
	require.NoError(t, err, "Error starting receiver 1")
	defer func() {
		require.NoError(t, r1.Shutdown(ctx), "Error shutting down receiver 1")
	}()

	err = r2.Start(ctx, nil)
	require.NoError(t, err, "Error starting receiver 2")
	defer func() {
		require.NoError(t, r2.Shutdown(ctx), "Error shutting down receiver 2")
	}()

	params.AssertFunc(t, logs)
}
