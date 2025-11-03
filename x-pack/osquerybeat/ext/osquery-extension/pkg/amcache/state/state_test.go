// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package state

import (
	"os"
//	"reflect"
	"testing"
	"strings"
	"time"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/testdata"	
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/filters"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	"github.com/osquery/osquery-go/plugin/table"
)

func TestCachingBehavior(t *testing.T) {
	log := logger.New(os.Stdout, true)
	tests := []struct {
		name string // description of this test case
		filePath string
		wantRecovered    bool
		wantErr      bool	
	}{
		//
		{
			name: "recovery test data", 
			filePath: testdata.GetRecoveryTestDataPathOrFatal(t),
			wantErr: false,
		},
		{
			name: "regular test data", 
			filePath: testdata.GetTestHivePathOrFatal(t),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new state with the test data
			newState := newAmcacheGlobalState(tt.filePath, defaultExpirationDuration)
			if newState == nil {
				t.Errorf("Expected newState to be initialized")
			}
			// Check the state configuration
			if newState.HivePath != tt.filePath {
				t.Errorf("%s: Expected hive path %s, got %s", tt.name, tt.filePath, newState.HivePath)
			}
			if newState.ExpirationDuration != defaultExpirationDuration {
				t.Errorf("%s: Expected expiration duration %s, got %s", tt.name, defaultExpirationDuration, newState.ExpirationDuration)
			}
			// Check the last updated time
			if !newState.LastUpdated.IsZero() {
				t.Errorf("%s: Expected lastUpdated to be zero initially, got %v", tt.name, newState.LastUpdated)
			}

			// Update the state
			err := newState.Update(log)
			if err != nil {
				t.Errorf("%s: Expected Update to succeed, got error: %v", tt.name, err)
			}
			if newState.LastUpdated.IsZero() {
				t.Errorf("%s: Expected lastUpdated to be set after Update, got %v", tt.name, newState.LastUpdated)
			}

			// Check the cache
			for _, table := range tables.AllAmcacheTables() {
				if len(newState.Cache[table.Name]) == 0 {
					t.Errorf("%s: Expected cache for table %s to be populated, got 0", tt.name, table.Name)
				}
			}

		})
	}
}

func TestGetCachedEntries(t *testing.T) {
	log := logger.New(os.Stdout, true)
	state := newAmcacheGlobalState(testdata.GetTestHivePathOrFatal(t), defaultExpirationDuration)
	if state == nil {
		t.Errorf("Expected newState to be initialized")
	}

	if !state.LastUpdated.IsZero() {
		t.Errorf("Expected lastUpdated to be zero initially, got %v", state.LastUpdated)
	}

	for _, table := range tables.AllAmcacheTables() {
		entries, err := state.GetCachedEntries(table, nil, log)
		if err != nil {
			t.Errorf("Expected GetCachedEntries to succeed, got error: %v", err)
		}
		if len(entries) == 0 {
			t.Errorf("Expected cache for table %s to be populated, got 0", table.Name)
		}
	}

	if state.LastUpdated.IsZero() {
		t.Errorf("Expected lastUpdated to be set after GetCachedEntries, got %v", state.LastUpdated)
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
			Operator: table.OperatorLike,
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
		name := entry.(*tables.ApplicationEntry).Name
		if !strings.Contains(name, "Microsoft") {
			t.Errorf("Expected entry %s to contain Microsoft", name)
		}
	}
}

func TestGetCachedEntriesForcesUpdate(t *testing.T) {
	log := logger.New(os.Stdout, true)
	state := newAmcacheGlobalState(testdata.GetTestHivePathOrFatal(t), 3 * time.Minute)

	if !state.IsExpired() {
		t.Errorf("Expected state to be expired, got %v", state.LastUpdated)
	}
	
	err := state.Update(log)
	if err != nil {
		t.Errorf("Expected Update to succeed, got error: %v", err)
	}
	if state.IsExpired() {
		t.Errorf("Expected state to be not be expired, got %v", state.LastUpdated)
	}
	lastUpdated := state.LastUpdated

	//rewind the last updated time by 6 minutes and make sure getCachedEntries forces an update
	state.LastUpdated = state.LastUpdated.Add(-6 * time.Minute)
	if !state.IsExpired() {
		t.Errorf("Expected state to be expired, got %v", state.LastUpdated)
	}

	entries, err := state.GetCachedEntries(*tables.GetAmcacheTableByName(tables.TableNameApplication), nil, log)
	if err != nil {
		t.Errorf("Expected GetCachedEntries to succeed, got error: %v", err)
	}
	if len(entries) == 0 {
		t.Errorf("Expected cache for table %s to be populated, got 0", tables.TableNameApplication)
	}
	if !state.LastUpdated.After(lastUpdated) {
		t.Errorf("Expected lastUpdated to be after the original, got %v", state.LastUpdated)
	}
}
