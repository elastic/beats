// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package state

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/filters"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

func getTestDataDirectory() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get current file path")
	}
	dir := filepath.Dir(currentFile)
	testDataDirectory := filepath.Join(dir, "..", "testdata")
	if _, err := os.Stat(testDataDirectory); os.IsNotExist(err) {
		return "", fmt.Errorf("test data directory does not exist: %w", err)
	}
	return testDataDirectory, nil
}

func getTestHivePath() (string, error) {
	testDataDirectory, err := getTestDataDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(testDataDirectory, "amcache.hve"), nil
}

func getRecoveryTestDataPath() (string, error) {
	testDataDirectory, err := getTestDataDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(testDataDirectory, "recovery_data", "Amcache.hve"), nil
}

func TestCachingBehavior(t *testing.T) {
	log := logger.New(os.Stdout, true)
	hivePath, err := getTestHivePath()
	if err != nil {
		t.Fatalf("failed to get test hive path: %v", err)
	}
	recoveryTestDataPath, err := getRecoveryTestDataPath()
	if err != nil {
		t.Fatalf("failed to get recovery test data path: %v", err)
	}
	tests := []struct {
		name          string // description of this test case
		filePath      string
		wantRecovered bool
		wantErr       bool
	}{
		{
			name:     "recovery test data",
			filePath: recoveryTestDataPath,
			wantErr:  false,
		},
		{
			name:     "regular test data",
			filePath: hivePath,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new state with the test data
			newState := newAmcacheState(tt.filePath, defaultExpirationDuration)

			// Check the state configuration
			assert.Equal(t, tt.filePath, newState.hivePath, "%s: Expected hive path %s, got %s", tt.name, tt.filePath, newState.hivePath)
			assert.Equal(t, defaultExpirationDuration, newState.expirationDuration, "%s: Expected expiration duration %s, got %s", tt.name, defaultExpirationDuration, newState.expirationDuration)

			// Update the state
			err := newState.updateLockHeld(log)
			assert.NoError(t, err, "%s: Expected Update to succeed, got error: %v", tt.name, err)

			// Check the cache
			for _, table := range tables.AllAmcacheTables() {
				assert.NotEmpty(t, newState.cache[table.Name], "%s: Expected cache for table %s to be populated, got 0", tt.name, table.Name)
			}
		})
	}
}

func TestGetCachedEntries(t *testing.T) {
	log := logger.New(os.Stdout, true)
	hivePath, err := getTestHivePath()
	if err != nil {
		t.Fatalf("failed to get test hive path: %v", err)
	}
	state := newAmcacheState(hivePath, defaultExpirationDuration)

	for _, table := range tables.AllAmcacheTables() {
		entries, err := state.GetCachedEntries(*tables.GetAmcacheTableByName(table.Name), nil, log)
		assert.NoError(t, err, "Expected GetCachedEntries to succeed, got error: %v", err)
		assert.NotEmpty(t, entries, "Expected cache for table %s to be populated, got 0", table.Name)
	}

	nonFilteredEntries, err := state.GetCachedEntries(*tables.GetAmcacheTableByName(tables.TableNameApplication), nil, log)
	assert.NoError(t, err, "Expected GetCachedEntries to succeed, got error: %v", err)
	assert.NotEmpty(t, nonFilteredEntries, "Expected cache for table %s to be populated, got 0", tables.TableNameApplication)

	filters := []filters.Filter{
		{
			ColumnName: "name",
			Operator:   table.OperatorLike,
			Expression: "%Microsoft%",
		},
	}
	filteredEntries, err := state.GetCachedEntries(*tables.GetAmcacheTableByName(tables.TableNameApplication), filters, log)
	assert.NoError(t, err, "Expected GetCachedEntries to succeed, got error: %v", err)
	assert.NotEmpty(t, filteredEntries, "Expected cache for table %s to be populated, got 0", tables.TableNameApplication)
	assert.Less(t, len(filteredEntries), len(nonFilteredEntries), "Expected less than %d entries, got %d", len(nonFilteredEntries), len(filteredEntries))

	for _, entry := range filteredEntries {
		appEntry, ok := entry.(*tables.ApplicationEntry)
		assert.True(t, ok, "Expected entry to be a ApplicationEntry, got %T", entry)
		assert.Contains(t, appEntry.Name, "Microsoft", "Expected entry %s to contain Microsoft", appEntry.Name)
	}
}

func TestGetCachedEntriesForcesUpdate(t *testing.T) {
	log := logger.New(os.Stdout, true)
	hivePath, err := getTestHivePath()
	if err != nil {
		t.Fatalf("failed to get test hive path: %v", err)
	}
	state := newAmcacheState(hivePath, 5*time.Second)

	state.lock.Lock()
	err = state.updateLockHeld(log)
	state.lock.Unlock()

	assert.NoError(t, err, "Expected Update to succeed, got error: %v", err)

	cacheExpired := false
	for _ = range 10 {
		state.lock.RLock()
		cache := state.cache
		state.lock.RUnlock()
		if cache == nil {
			cacheExpired = true
			break
		}
		time.Sleep(1 * time.Second)
	}
	assert.True(t, cacheExpired, "Expected cache to be expired, got %v", cacheExpired)

	entries, err := state.GetCachedEntries(*tables.GetAmcacheTableByName(tables.TableNameApplication), nil, log)
	assert.NoError(t, err, "Expected GetCachedEntries to succeed, got error: %v", err)
	assert.NotEmpty(t, entries, "Expected cache for table %s to be populated, got 0", tables.TableNameApplication)
}
