// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package azure

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/memlog"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure/cursor"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

// newTestMetricSetWithCursor builds a minimal MetricSet with an in-memory cursor
// store for testing cursor methods without standing up real Azure clients.
func newTestMetricSetWithCursor(t *testing.T, lookbackWindow time.Duration) (*MetricSet, *statestore.Registry) {
	t.Helper()
	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home: tmpDir, Config: tmpDir, Data: tmpDir, Logs: tmpDir,
	}
	logger := logp.NewNopLogger()
	dataPath := beatPaths.Resolve(paths.Data, "azure-cursor")
	reg, err := memlog.New(logger.Named("memlog"), memlog.Settings{
		Root: dataPath, FileMode: 0o600,
	})
	require.NoError(t, err)
	registry := statestore.NewRegistry(reg)
	t.Cleanup(func() { registry.Close() })

	store, err := cursor.NewStoreFromRegistry(registry, logger)
	require.NoError(t, err)

	ms := &MetricSet{
		cursorStore:    store,
		cursorKey:      "test-key",
		lookbackWindow: lookbackWindow,
		cursorLogger:   logger,
	}
	return ms, registry
}

func TestComputeLookbackStart(t *testing.T) {
	referenceTime, _ := time.Parse(time.RFC3339, "2024-07-30T19:00:00Z")
	lookbackWindow := 10 * time.Minute

	t.Run("cold start — no cursor in store returns nil", func(t *testing.T) {
		ms, _ := newTestMetricSetWithCursor(t, lookbackWindow)
		defer ms.Close()
		result := ms.computeLookbackStart(referenceTime)
		assert.Nil(t, result)
	})

	t.Run("gap within window returns cursor time", func(t *testing.T) {
		ms, _ := newTestMetricSetWithCursor(t, lookbackWindow)
		defer ms.Close()

		// 7 minutes ago — within the 10m lookback window
		lastEnd, _ := time.Parse(time.RFC3339, "2024-07-30T18:53:00Z")
		state := &cursor.State{
			Version: cursor.StateVersion, LastCollectionEnd: lastEnd, UpdatedAt: time.Now(),
		}
		require.NoError(t, ms.cursorStore.Save(ms.cursorKey, state))

		result := ms.computeLookbackStart(referenceTime)
		require.NotNil(t, result)
		assert.True(t, lastEnd.Equal(*result))
	})

	t.Run("gap beyond window returns nil", func(t *testing.T) {
		ms, _ := newTestMetricSetWithCursor(t, lookbackWindow)
		defer ms.Close()

		// 15 minutes ago — beyond the 10m lookback window
		lastEnd, _ := time.Parse(time.RFC3339, "2024-07-30T18:45:00Z")
		state := &cursor.State{
			Version: cursor.StateVersion, LastCollectionEnd: lastEnd, UpdatedAt: time.Now(),
		}
		require.NoError(t, ms.cursorStore.Save(ms.cursorKey, state))

		result := ms.computeLookbackStart(referenceTime)
		assert.Nil(t, result)
	})

	t.Run("nil cursorStore returns nil", func(t *testing.T) {
		ms := &MetricSet{cursorStore: nil, lookbackWindow: lookbackWindow}
		result := ms.computeLookbackStart(referenceTime)
		assert.Nil(t, result)
	})

	// With latency=5m the cursor is saved at referenceTime-latency, so the
	// effective downtime budget should still be the full lookback_window, not
	// lookback_window - latency.
	t.Run("latency: cursor within window relative to endTime returns cursor time", func(t *testing.T) {
		ms, _ := newTestMetricSetWithCursor(t, lookbackWindow)
		defer ms.Close()
		ms.latency = 5 * time.Minute

		// referenceTime=19:00, latency=5m → endTime=18:55
		// cursor=18:50 is 5m before endTime, well within the 10m window
		// (it is 10m before referenceTime — old code would accept it; new code also accepts it)
		lastEnd, _ := time.Parse(time.RFC3339, "2024-07-30T18:50:00Z")
		state := &cursor.State{Version: cursor.StateVersion, LastCollectionEnd: lastEnd, UpdatedAt: time.Now()}
		require.NoError(t, ms.cursorStore.Save(ms.cursorKey, state))

		result := ms.computeLookbackStart(referenceTime)
		require.NotNil(t, result)
		assert.True(t, lastEnd.Equal(*result))
	})

	t.Run("latency: cursor 12m before referenceTime is within 10m window relative to endTime", func(t *testing.T) {
		ms, _ := newTestMetricSetWithCursor(t, lookbackWindow)
		defer ms.Close()
		ms.latency = 5 * time.Minute

		// referenceTime=19:00, latency=5m → endTime=18:55, minStart=18:45
		// cursor=18:48 is 12m before referenceTime but only 7m before endTime → within window
		// old code (minStart=referenceTime-10m=18:50) would have rejected this cursor
		lastEnd, _ := time.Parse(time.RFC3339, "2024-07-30T18:48:00Z")
		state := &cursor.State{Version: cursor.StateVersion, LastCollectionEnd: lastEnd, UpdatedAt: time.Now()}
		require.NoError(t, ms.cursorStore.Save(ms.cursorKey, state))

		result := ms.computeLookbackStart(referenceTime)
		require.NotNil(t, result, "cursor within lookback_window relative to endTime should be accepted")
		assert.True(t, lastEnd.Equal(*result))
	})

	t.Run("latency: cursor beyond window relative to endTime returns nil", func(t *testing.T) {
		ms, _ := newTestMetricSetWithCursor(t, lookbackWindow)
		defer ms.Close()
		ms.latency = 5 * time.Minute

		// referenceTime=19:00, latency=5m → endTime=18:55, minStart=18:45
		// cursor=18:40 is 15m before endTime → beyond the 10m window
		lastEnd, _ := time.Parse(time.RFC3339, "2024-07-30T18:40:00Z")
		state := &cursor.State{Version: cursor.StateVersion, LastCollectionEnd: lastEnd, UpdatedAt: time.Now()}
		require.NoError(t, ms.cursorStore.Save(ms.cursorKey, state))

		result := ms.computeLookbackStart(referenceTime)
		assert.Nil(t, result)
	})
}

func TestUpdateCursorKey(t *testing.T) {
	const (
		metricsetName  = "storage"
		subscriptionID = "sub-123"
	)

	t.Run("key changes when resources are updated", func(t *testing.T) {
		ms, _ := newTestMetricSetWithCursor(t, 30*time.Minute)
		defer ms.Close()

		initialKey := ms.cursorKey

		resources := []ResourceConfig{
			{Query: "resourceType eq 'Microsoft.Storage/storageAccounts'"},
		}
		ms.UpdateCursorKey(metricsetName, subscriptionID, resources)

		assert.NotEqual(t, initialKey, ms.cursorKey,
			"cursor key should change after UpdateCursorKey is called with non-empty resources")
	})

	t.Run("two metricsets with equivalent configs share the same key", func(t *testing.T) {
		// Simulate a default-config storage metricset: NewMetricSet runs with
		// empty Resources, then storage.New() injects the default resource query.
		msDefault, _ := newTestMetricSetWithCursor(t, 30*time.Minute)
		defer msDefault.Close()

		defaultResources := []ResourceConfig{
			{Query: "resourceType eq 'Microsoft.Storage/storageAccounts'"},
		}
		msDefault.UpdateCursorKey(metricsetName, subscriptionID, defaultResources)

		// Simulate an explicit-config storage metricset: NewMetricSet runs with
		// Resources already matching the default, no update needed — but the key
		// should equal the one produced by the default path.
		msExplicit, _ := newTestMetricSetWithCursor(t, 30*time.Minute)
		defer msExplicit.Close()
		msExplicit.UpdateCursorKey(metricsetName, subscriptionID, defaultResources)

		assert.Equal(t, msDefault.cursorKey, msExplicit.cursorKey,
			"default and equivalent explicit storage configs should produce the same cursor key")
	})

	t.Run("no-op when cursorStore is nil", func(t *testing.T) {
		ms := &MetricSet{cursorStore: nil, lookbackWindow: 10 * time.Minute}
		before := ms.cursorKey
		ms.UpdateCursorKey("storage", "sub-123", []ResourceConfig{{Query: "resourceType eq 'X'"}})
		assert.Equal(t, before, ms.cursorKey)
	})
}

func TestUpdateCursor(t *testing.T) {
	endTime, _ := time.Parse(time.RFC3339, "2024-07-30T19:00:00Z")

	t.Run("saves cursor on happy path", func(t *testing.T) {
		ms, _ := newTestMetricSetWithCursor(t, 10*time.Minute)
		defer ms.Close()

		ms.updateCursor(endTime)

		state, err := ms.cursorStore.Load(ms.cursorKey)
		require.NoError(t, err)
		require.NotNil(t, state)
		assert.True(t, endTime.Equal(state.LastCollectionEnd))
	})

	t.Run("nil cursorStore is a no-op", func(t *testing.T) {
		ms := &MetricSet{cursorStore: nil}
		// Must not panic
		ms.updateCursor(endTime)
	})
}
