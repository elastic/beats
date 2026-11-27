// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap/zaptest"

	"github.com/elastic/beats/v7/pkg/mapstr"
)

// TestBeatMetricsDiagnosticIncludesHeartbeatMetrics checks that the receiver's
// beat_metrics.json diagnostic contains the heartbeat metrics that the process
// runtime reports, even though they live in the process-global registry.
func TestBeatMetricsDiagnosticIncludesHeartbeatMetrics(t *testing.T) {
	factory := NewFactoryWithSettings(Settings{Home: t.TempDir()})
	set := receiver.Settings{
		ID:     component.NewIDWithName(factory.Type(), "diagnostics"),
		Logger: zaptest.NewLogger(t),
	}

	// The monitor only needs to be valid so that a scheduler is created; the
	// target is never expected to answer.
	cfg := staticMonitorConfig(t, []map[string]any{{
		"type":     "http",
		"id":       "diagnostics",
		"schedule": "@every 1h",
		"urls":     []string{"http://127.0.0.1:0"},
	}})
	rec, err := factory.CreateLogs(t.Context(), set, cfg, consumertest.NewNop())
	require.NoError(t, err, "creating the receiver should succeed")

	diagExt := &fakeDiagnosticExtension{hooks: map[string]func() []byte{}}
	require.NoError(t, rec.Start(t.Context(), &hostWithExtension{ext: diagExt}), "starting the receiver should succeed")
	t.Cleanup(func() {
		require.NoError(t, rec.Shutdown(context.Background()), "shutting down the receiver should succeed")
	})

	// The registry is process-global and shared with other tests in the
	// package, so only the structure is asserted, not the values.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var snapshot mapstr.M
		if !assert.NoError(c, json.Unmarshal([]byte(diagExt.run(t, "beat_metrics.json")), &snapshot), "beat_metrics.json should be valid JSON") {
			return
		}
		for _, key := range []string{
			"heartbeat.scheduler.jobs.active",
			"heartbeat.scheduler.jobs.missed_deadline",
			"heartbeat.scheduler.tasks.active",
			"heartbeat.scheduler.tasks.waiting",
			"heartbeat.http.monitor_starts",
			"heartbeat.http.endpoint_starts",
			"heartbeat.config.module.running",
			// The receiver's own metrics must still be there.
			"beat.info",
			"libbeat.pipeline",
		} {
			has, err := snapshot.HasKey(key)
			assert.NoError(c, err, "looking up %s should succeed", key)
			assert.True(c, has, "beat_metrics.json should contain %s, got %v", key, snapshot)
		}
	}, 30*time.Second, 100*time.Millisecond)
}

// hostWithExtension is a collectorHost with a diagnostics extension.
type hostWithExtension struct {
	collectorHost
	ext component.Component
}

func (h *hostWithExtension) GetExtensions() map[component.ID]component.Component {
	return map[component.ID]component.Component{component.MustNewID("elastic_diagnostics"): h.ext}
}

// fakeDiagnosticExtension records the diagnostic hooks registered with it.
type fakeDiagnosticExtension struct {
	component.StartFunc
	component.ShutdownFunc

	mx    sync.Mutex
	hooks map[string]func() []byte
}

func (e *fakeDiagnosticExtension) RegisterDiagnosticHook(_ string, _ string, filename string, _ string, hook func() []byte) {
	e.mx.Lock()
	defer e.mx.Unlock()
	e.hooks[filename] = hook
}

func (e *fakeDiagnosticExtension) run(t *testing.T, filename string) string {
	e.mx.Lock()
	hook, ok := e.hooks[filename]
	e.mx.Unlock()
	require.True(t, ok, "no diagnostic hook for %s", filename)
	return string(hook())
}
