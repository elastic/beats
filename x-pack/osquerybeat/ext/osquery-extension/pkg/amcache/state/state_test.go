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
	"strings"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"

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
			if newState.hivePath != tt.filePath {
				t.Errorf("%s: Expected hive path %s, got %s", tt.name, tt.filePath, newState.hivePath)
			}
			if newState.expirationDuration != defaultExpirationDuration {
				t.Errorf("%s: Expected expiration duration %s, got %s", tt.name, defaultExpirationDuration, newState.expirationDuration)
			}
			// Check the last updated time
			if !newState.lastUpdated.IsZero() {
				t.Errorf("%s: Expected lastUpdated to be zero initially, got %v", tt.name, newState.lastUpdated)
			}

			// Update the state
			err := newState.updateLockHeld(log)
			if err != nil {
				t.Errorf("%s: Expected Update to succeed, got error: %v", tt.name, err)
			}
			if newState.lastUpdated.IsZero() {
				t.Errorf("%s: Expected lastUpdated to be set after Update, got %v", tt.name, newState.lastUpdated)
			}

			// Check the cache
			for _, table := range tables.AllAmcacheTables() {
				if len(newState.cache[table.Name]) == 0 {
					t.Errorf("%s: Expected cache for table %s to be populated, got 0", tt.name, table.Name)
				}
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

	if !state.lastUpdated.IsZero() {
		t.Errorf("Expected lastUpdated to be zero initially, got %v", state.lastUpdated)
	}

	for _, table := range tables.AllAmcacheTables() {
		entries, err := state.GetCachedEntries(*tables.GetAmcacheTableByName(table.Name), nil, log)
		if err != nil {
			t.Errorf("Expected GetCachedEntries to succeed, got error: %v", err)
		}
		if len(entries) == 0 {
			t.Errorf("Expected cache for table %s to be populated, got 0", table.Name)
		}
	}

	if state.lastUpdated.IsZero() {
		t.Errorf("Expected lastUpdated to be set after GetCachedEntries, got %v", state.lastUpdated)
	}

	nonFilteredEntries, err := state.GetCachedEntries(*tables.GetAmcacheTableByName(tables.TableNameApplication), nil, log)
	if err != nil {
		t.Errorf("Expected GetCachedEntries to succeed, got error: %v", err)
	}
	if len(nonFilteredEntries) == 0 {
		t.Errorf("Expected cache for table %s to be populated, got 0", tables.TableNameApplication)
	}

	filters := []filters.Filter{
		{
			ColumnName: "name",
			Operator:   table.OperatorLike,
			Expression: "%Microsoft%",
		},
	}
	filteredEntries, err := state.GetCachedEntries(*tables.GetAmcacheTableByName(tables.TableNameApplication), filters, log)
	if err != nil {
		t.Errorf("Expected GetCachedEntries to succeed, got error: %v", err)
	}
	if len(filteredEntries) == 0 {
		t.Errorf("Expected cache for table %s to be populated, got 0", tables.TableNameApplication)
	}
	if len(filteredEntries) >= len(nonFilteredEntries) {
		t.Errorf("Expected less than %d entries, got %d", len(nonFilteredEntries), len(filteredEntries))
	}

	for _, entry := range filteredEntries {
		appEntry, ok := entry.(*tables.ApplicationEntry)
		if !ok {
			t.Errorf("Expected entry to be a ApplicationEntry, got %T", entry)
			continue
		}
		name := appEntry.Name
		if !strings.Contains(name, "Microsoft") {
			t.Errorf("Expected entry %s to contain Microsoft", name)
		}
	}
}

func TestGetCachedEntriesForcesUpdate(t *testing.T) {
	log := logger.New(os.Stdout, true)
	hivePath, err := getTestHivePath()
	if err != nil {
		t.Fatalf("failed to get test hive path: %v", err)
	}
	state := newAmcacheState(hivePath, 3*time.Minute)

	isStateExpired := func() bool {
		state.lock.RLock()
		defer state.lock.RUnlock()
		return time.Since(state.lastUpdated) > state.expirationDuration
	}

	if !isStateExpired() {
		t.Errorf("Expected state to be expired, got %v", state.lastUpdated)
	}

	state.lock.Lock()
	err = state.updateLockHeld(log)
	state.lock.Unlock()

	if err != nil {
		t.Errorf("Expected Update to succeed, got error: %v", err)
	}
	if isStateExpired() {
		t.Errorf("Expected state to be not be expired, got %v", state.lastUpdated)
	}
	lastUpdated := state.lastUpdated

	//rewind the last updated time by 6 minutes and make sure getCachedEntries forces an update
	state.lastUpdated = state.lastUpdated.Add(-6 * time.Minute)
	if !isStateExpired() {
		t.Errorf("Expected state to be expired, got %v", state.lastUpdated)
	}

	entries, err := state.GetCachedEntries(*tables.GetAmcacheTableByName(tables.TableNameApplication), nil, log)
	if err != nil {
		t.Errorf("Expected GetCachedEntries to succeed, got error: %v", err)
	}
	if len(entries) == 0 {
		t.Errorf("Expected cache for table %s to be populated, got 0", tables.TableNameApplication)
	}
	if !state.lastUpdated.After(lastUpdated) {
		t.Errorf("Expected lastUpdated to be after the original, got %v", state.lastUpdated)
	}
}
