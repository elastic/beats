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
	Name    string
	Config  component.Config
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

		var receiverSettings receiver.Settings

		// Replicate the behavior of the collector logger
		receiverCore := core.
			With([]zapcore.Field{
				zap.String("otelcol.component.id", rc.Name),
				zap.String("otelcol.component.kind", "Receiver"),
				zap.String("otelcol.signals", "logs"),
			})

		receiverSettings.Logger = zap.New(receiverCore)
		receiverSettings.ID = component.NewIDWithName(rc.Factory.Type(), rc.Name)

		logConsumer, err := consumer.NewLogs(func(ctx context.Context, ld plog.Logs) error {
			for i := 0; i < ld.ResourceLogs().Len(); i++ {
				rl := ld.ResourceLogs().At(i)
				for j := 0; j < rl.ScopeLogs().Len(); j++ {
					sl := rl.ScopeLogs().At(j)
					for k := 0; k < sl.LogRecords().Len(); k++ {
						log := sl.LogRecords().At(k)
						logsMu.Lock()
						logs[rc.Name] = append(logs[rc.Name], log.Body().Map().AsRaw())
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

	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		logsMu.Lock()
		defer logsMu.Unlock()

		// Ensure the logger fields from the otel collector are present in the logs.

		for _, zl := range zapLogs.All() {
			require.Contains(t, zl.ContextMap(), "otelcol.component.id")
			require.Equal(t, zl.ContextMap()["otelcol.component.kind"], "Receiver")
			require.Equal(t, zl.ContextMap()["otelcol.signals"], "logs")
			break
		}

		params.AssertFunc(ct, logs, zapLogs)
	}, 2*time.Minute, 100*time.Millisecond,
		"timeout waiting for logger fields from the OTel collector are present in the logs and other assertions to be met")
}
