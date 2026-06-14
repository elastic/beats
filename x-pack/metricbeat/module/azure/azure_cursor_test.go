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
