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
	"context"
	"sync"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

type ReceiverConfig struct {
	// Name is the unique identifier for the component
	Name string
	// Beat is the name of the Beat that is running as a receiver
	Beat string
	// Config is the configuration for the receiver component
	Config component.Config
	// Factory is the factory to instantiate the receiver
	Factory receiver.Factory
}

type CheckReceiversParams struct {
	T *testing.T
	// Receivers is a list of receiver configurations to create.
	Receivers []ReceiverConfig
	// AssertFunc is a function that asserts the test conditions.
	// The function is called periodically until the assertions are met or the timeout is reached.
	AssertFunc func(t *assert.CollectT, logs map[string][]mapstr.M, zapLogs *observer.ObservedLogs)
}

// CheckReceivers creates receivers using the provided configuration.
func CheckReceivers(params CheckReceiversParams) {
	t := params.T
	ctx := t.Context()

	var logsMu sync.Mutex
	logs := make(map[string][]mapstr.M)

	zapCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		&zaptest.Discarder{},
		zapcore.DebugLevel,
	)
	observed, zapLogs := observer.New(zapcore.DebugLevel)

	core := zapcore.NewTee(zapCore, observed)

	createReceiver := func(t *testing.T, rc ReceiverConfig) receiver.Logs {
		t.Helper()

		require.NotEmpty(t, rc.Name, "receiver name must not be empty")
		require.NotEmpty(t, rc.Beat, "receiver beat must not be empty")

		var receiverSettings receiver.Settings

		// Replicate the behavior of the collector logger
		receiverCore := core.
			With([]zapcore.Field{
				zap.String("otelcol.component.id", rc.Name),
				zap.String("otelcol.component.kind", "receiver"),
				zap.String("otelcol.signal", "logs"),
			})

		receiverSettings.Logger = zap.New(receiverCore)
		receiverSettings.ID = component.NewIDWithName(rc.Factory.Type(), rc.Name)

		logConsumer, err := consumer.NewLogs(func(ctx context.Context, ld plog.Logs) error {
			for _, rl := range ld.ResourceLogs().All() {
				for _, sl := range rl.ScopeLogs().All() {
					for _, lr := range sl.LogRecords().All() {
						logsMu.Lock()
						logs[rc.Name] = append(logs[rc.Name], lr.Body().Map().AsRaw())
						logsMu.Unlock()
					}
				}
			}
			return nil
		})
		assert.NoErrorf(t, err, "Error creating log consumer for %q", rc.Name)

		r, err := rc.Factory.CreateLogs(ctx, receiverSettings, rc.Config, logConsumer)
		assert.NoErrorf(t, err, "Error creating receiver %q", rc.Name)
		return r
	}

	// Replicate the collector behavior to instantiate components first and then start them.
	var receivers []receiver.Logs
	for _, rec := range params.Receivers {
		receivers = append(receivers, createReceiver(t, rec))
	}

	for i, r := range receivers {
		err := r.Start(ctx, nil)
		require.NoErrorf(t, err, "Error starting receiver %d", i)
		defer func() {
			require.NoErrorf(t, r.Shutdown(ctx), "Error shutting down receiver %d", i)
		}()
	}

	t.Cleanup(func() {
		if t.Failed() {
			logsMu.Lock()
			defer logsMu.Unlock()
			t.Logf("Ingested Logs: %v", logs)
		}
	})

	beatForCompID := func(compID string) string {
		for _, rec := range params.Receivers {
			if rec.Name == compID {
				return rec.Beat
			}
		}

		return ""
	}

	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		logsMu.Lock()
		defer logsMu.Unlock()

		// Ensure the logger fields from the otel collector are present
		for _, zl := range zapLogs.All() {
			require.Contains(t, zl.ContextMap(), "otelcol.component.kind")
			require.Equal(t, "receiver", zl.ContextMap()["otelcol.component.kind"])
			require.Contains(t, zl.ContextMap(), "otelcol.signal")
			require.Equal(t, "logs", zl.ContextMap()["otelcol.signal"])
			require.Contains(t, zl.ContextMap(), "otelcol.component.id")
			compID, ok := zl.ContextMap()["otelcol.component.id"].(string)
			require.True(t, ok, "otelcol.component.id should be a string")
			require.Contains(t, zl.ContextMap(), "service.name")
			require.Equal(t, beatForCompID(compID), zl.ContextMap()["service.name"])
			break
		}

		params.AssertFunc(ct, logs, zapLogs)
	}, 2*time.Minute, 100*time.Millisecond,
		"timeout waiting for logger fields from the OTel collector are present in the logs and other assertions to be met")
}
