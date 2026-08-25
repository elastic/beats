// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
)

// TestSchedulerGroup checks the scheduler group each receiver hands to Heartbeat.
// Whether Heartbeats of a group then actually share concurrency limits is
// covered by the heartbeat/hbscheduler tests.
func TestSchedulerGroup(t *testing.T) {
	// A receiver that is shut down without ever being started must still give up
	// the scheduler it acquired.
	defer oteltest.VerifyNoLeaks(t)

	newReceiver := func(t *testing.T, name, schedulerGroup string) *observer.ObservedLogs {
		t.Helper()

		observed, logs := observer.New(zapcore.DebugLevel)
		factory := NewFactoryWithSettings(Settings{Home: t.TempDir()})

		set := receiver.Settings{}
		set.ID = component.NewIDWithName(factory.Type(), name)
		set.Logger = zap.New(observed)

		cfg := &Config{
			SchedulerGroup: schedulerGroup,
			Beatconfig: map[string]any{
				"heartbeat": map[string]any{
					"monitors": []map[string]any{
						{
							"type":     "tcp",
							"id":       "test-tcp-" + name,
							"schedule": "@every 60s",
							"hosts":    []string{"localhost:0"},
							"enabled":  true,
						},
					},
				},
				"logging": map[string]any{
					"level":     "debug",
					"selectors": []string{"*"},
				},
				"path.home": t.TempDir(),
			},
		}

		r, err := factory.CreateLogs(t.Context(), set, cfg, consumertest.NewNop())
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, r.Shutdown(t.Context())) })

		return logs
	}

	// groupsOf returns the scheduler groups the receiver acquired a scheduler for.
	groupsOf := func(t *testing.T, logs *observer.ObservedLogs) []string {
		t.Helper()

		var groups []string
		for _, entry := range logs.All() {
			if group, ok := entry.ContextMap()["scheduler_group"]; ok {
				groups = append(groups, group.(string))
			}
		}
		require.NotEmpty(t, groups, "receiver logged no scheduler group")
		return groups
	}

	t.Run("defaults to the receiver id", func(t *testing.T) {
		logs := newReceiver(t, "no-group", "")

		for _, group := range groupsOf(t, logs) {
			assert.Equal(t, "heartbeatreceiver/no-group", group)
		}
	})

	t.Run("uses the configured group", func(t *testing.T) {
		const group = "synthetics/synthetics-browser-default"
		logs := newReceiver(t, "with-group", group)

		for _, got := range groupsOf(t, logs) {
			assert.Equal(t, group, got)
		}
	})
}
