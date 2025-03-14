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
	Name   string
	Config component.Config
}

type CheckReceiversParams struct {
	T *testing.T
	// Factory that allows to create a receiver.
	Factory receiver.Factory
	// Receivers is a list of receiver configurations to create.
	Receivers []ReceiverConfig
	// AssertFunc is a function that asserts the test conditions.
	// The function is called periodically until it returns true which ends the test.
	AssertFunc func(t *testing.T, logs map[string][]mapstr.M, zapLogs *observer.ObservedLogs) bool
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

	createReceiver := func(t *testing.T, name string, cfg component.Config) receiver.Logs {
		t.Helper()

		var receiverSettings receiver.Settings

		// Replicate the behavior of the collector logger
		receiverCore := core.
			With([]zapcore.Field{
				zap.String("name", name),
				zap.String("kind", "receiver"),
				zap.String("data_type", "logs"),
			})

		receiverSettings.Logger = zap.New(receiverCore)

		logConsumer, err := consumer.NewLogs(func(ctx context.Context, ld plog.Logs) error {
			for i := 0; i < ld.ResourceLogs().Len(); i++ {
				rl := ld.ResourceLogs().At(i)
				for j := 0; j < rl.ScopeLogs().Len(); j++ {
					sl := rl.ScopeLogs().At(j)
					for k := 0; k < sl.LogRecords().Len(); k++ {
						log := sl.LogRecords().At(k)
						logsMu.Lock()
						logs[name] = append(logs[name], log.Body().Map().AsRaw())
						logsMu.Unlock()
					}
				}
			}
			return nil
		})
		assert.NoErrorf(t, err, "Error creating log consumer for %q", name)

		r, err := params.Factory.CreateLogs(ctx, receiverSettings, cfg, logConsumer)
		assert.NoErrorf(t, err, "Error creating receiver %q", name)
		return r
	}

	// Replicate the collector behavior to instantiate components first and then start them.
	var receivers []receiver.Logs
	for _, rec := range params.Receivers {
		receivers = append(receivers, createReceiver(t, rec.Name, rec.Config))
	}

	for i, r := range receivers {
		i++
		err := r.Start(ctx, nil)
		require.NoErrorf(t, err, "Error starting receiver %d", i)
		defer func() {
			require.NoErrorf(t, r.Shutdown(ctx), "Error shutting down receiver %d", i)
		}()
	}

	require.Eventually(t, func() bool {
		logsMu.Lock()
		defer logsMu.Unlock()

		// Ensure the logger fields from the otel collector are present in the logs.
		for _, zl := range zapLogs.All() {
			require.Contains(t, zl.ContextMap(), "name")
			require.Equal(t, zl.ContextMap()["kind"], "receiver")
			require.Equal(t, zl.ContextMap()["data_type"], "logs")
			break
		}

		return params.AssertFunc(t, logs, zapLogs)
	}, time.Minute, 100*time.Millisecond, "timeout waiting for assertion to pass")
}
