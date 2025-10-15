// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
	"context"
	"testing"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/testdata"
	"www.velocidex.com/golang/regparser"
)

func GetFilteredEntries(getEntriesFunc func(*regparser.Registry) (map[string][]Entry, error), hashes ...string) []Entry {
	registry, err := LoadRegistry(testdata.GetTestHivePathOrFatal(nil))
	if err != nil {
		return nil
	}
	entriesMap, err := getEntriesFunc(registry)
	if err != nil {
		return nil
	}
	if len(hashes) == 0 {
		var allEntries []Entry
		for _, entries := range entriesMap {
			allEntries = append(allEntries, entries...)
		}
		return allEntries
	} else {
		var filteredEntries []Entry
		for _, hash := range hashes {
			if entries, ok := entriesMap[hash]; ok {
				filteredEntries = append(filteredEntries, entries...)
			}
		}
		return filteredEntries
	}
}

type MockGlobalState struct{}
func (m *MockGlobalState) GetApplicationEntries(hashes ...string) []Entry {
	return GetFilteredEntries(GetApplicationEntriesFromRegistry, hashes...)
}
func (m *MockGlobalState) GetApplicationFileEntries(hashes ...string) []Entry {
	return GetFilteredEntries(GetApplicationFileEntriesFromRegistry, hashes...)
}
func (m *MockGlobalState) GetApplicationShortcutEntries(hashes ...string) []Entry {
	return GetFilteredEntries(GetApplicationShortcutEntriesFromRegistry, hashes...)
}
func (m *MockGlobalState) GetDriverBinaryEntries(hashes ...string) []Entry {
	return GetFilteredEntries(GetDriverBinaryEntriesFromRegistry, hashes...)
}
func (m *MockGlobalState) GetDevicePnpEntries(hashes ...string) []Entry {
	return GetFilteredEntries(GetDevicePnpEntriesFromRegistry, hashes...)
}


func TestTables(t *testing.T) {
	mockState := &MockGlobalState{}
	generateFuncs := []struct {
		name         string
		generateFunc table.GenerateFunc
		columns      []table.ColumnDefinition
	}{
		{"amcache_application", ApplicationGenerateFunc(mockState), ApplicationColumns()},
		{"amcache_application_file", ApplicationFileGenerateFunc(mockState), ApplicationFileColumns()},
		{"amcache_application_shortcut", ApplicationShortcutGenerateFunc(mockState), ApplicationShortcutColumns()},
		{"amcache_driver_binary", DriverBinaryGenerateFunc(mockState), DriverBinaryColumns()},
		{"amcache_device_pnp", DevicePnpGenerateFunc(mockState), DevicePnpColumns()},
	}

	for _, gf := range generateFuncs {
		t.Run(gf.name, func(t *testing.T) {
			rows, err := gf.generateFunc(context.Background(), table.QueryContext{})
			if err != nil {
				t.Fatalf("Error generating rows for %s: %v", gf.name, err)
			}
			if len(rows) == 0 {
				t.Fatalf("No rows returned for %s", gf.name)
			}
			for _, row := range rows {
				for _, column := range gf.columns {
					if _, ok := row[column.Name]; !ok {
						t.Errorf("Expected column %s in row, but not found", column.Name)
					}
				}
			}
		})
	}
}


