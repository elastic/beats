// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// TestMonitorRunLogsUseOwnReceiverLogger checks that the "Monitor finished"
// run log of each receiver goes to that receiver's own logger, not to a
// process-wide one shared by every receiver.
func TestMonitorRunLogsUseOwnReceiverLogger(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	type rcv struct {
		name string
		logs *observer.ObservedLogs
		rec  receiver.Logs
	}

	// create only builds the receiver; both are created before either is
	// started to avoid a race on the global timestamp precision setting.
	create := func(name string) rcv {
		core, logs := observer.New(zapcore.DebugLevel)
		monitor := map[string]any{
			"id":       name,
			"name":     name,
			"type":     "http",
			"urls":     []string{srv.URL},
			"schedule": "@every 1h",
			"enabled":  true,
		}
		cfg := &Config{Beatconfig: map[string]any{
			"heartbeat":               map[string]any{"monitors": []any{monitor}},
			"queue.mem.flush.timeout": "0s",
			"path.home":               t.TempDir(),
		}}
		factory := NewFactoryWithSettings(Settings{Home: t.TempDir()})
		set := receiver.Settings{}
		set.ID = component.NewIDWithName(factory.Type(), name)
		set.Logger = zap.New(core)

		rec, err := factory.CreateLogs(t.Context(), set, cfg, consumertest.NewNop())
		require.NoError(t, err, "creating receiver %s", name)
		return rcv{name: name, logs: logs, rec: rec}
	}

	hasRunLog := func(r rcv) bool {
		for _, e := range r.logs.All() {
			if strings.Contains(e.Message, "Monitor finished") || strings.Contains(e.Message, "error gathering information") {
				return true
			}
		}
		return false
	}

	first, second := create("monitor-a"), create("monitor-b")
	for _, r := range []rcv{first, second} {
		require.NoError(t, r.rec.Start(t.Context(), componenttest.NewNopHost()), "starting receiver %s", r.name)
		t.Cleanup(func() { assert.NoError(t, r.rec.Shutdown(context.Background()), "stopping receiver %s", r.name) })
	}

	for _, r := range []rcv{first, second} {
		assert.Eventuallyf(t, func() bool { return hasRunLog(r) },
			30*time.Second, 100*time.Millisecond, "run log of %s must go to its own receiver logger", r.name)
	}
}
