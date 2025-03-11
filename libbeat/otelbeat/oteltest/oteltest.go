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
	AssertFunc func(t *testing.T, logs map[string][]mapstr.M, zapLogs []byte) bool
}

// CheckReceivers creates receivers using the provided configuration.
func CheckReceivers(params CheckReceiversParams) {
	t := params.T
	var logsMu sync.Mutex
	logs := make(map[string][]mapstr.M)

	var zapLogs bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&zapLogs),
		zapcore.DebugLevel)

	ctx := context.Background()
	createReceiver := func(t *testing.T, name string, cfg component.Config) receiver.Logs {
		t.Helper()

		var receiverSettings receiver.Settings
		receiverSettings.Logger = zap.New(core).Named(name)

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

		t.Cleanup(func() {
			if t.Failed() {
				t.Logf("Logs for %q: %s\n", name, zapLogs.String())
			}
		})

		r, err := params.Factory.CreateLogs(ctx, receiverSettings, cfg, logConsumer)
		assert.NoErrorf(t, err, "Error creating receiver %q", name)
		return r
	}

	for _, rec := range params.Receivers {
		r := createReceiver(t, rec.Name, rec.Config)
		err := r.Start(ctx, nil)
		require.NoErrorf(t, err, "Error starting receiver %q", rec.Name)
		defer func() {
			require.NoErrorf(t, r.Shutdown(ctx), "Error shutting down receiver %q", rec.Name)
		}()
	}

	require.Eventually(t, func() bool {
		logsMu.Lock()
		defer logsMu.Unlock()

		return params.AssertFunc(t, logs, zapLogs.Bytes())
	}, time.Minute, 100*time.Millisecond, "timeout waiting for assertion to pass")
}
