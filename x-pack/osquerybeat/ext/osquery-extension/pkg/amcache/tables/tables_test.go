// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
	"context"
	"os"
	"testing"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/testdata"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/filters"
)

func GetFilteredEntries(amcacheTable AmcacheTable, filters []filters.Filter) []Entry {
	registry, err := LoadRegistry(testdata.GetTestHivePathOrFatal(nil))
	if err != nil {
		return nil
	}
	entries, err := GetEntriesFromRegistry(amcacheTable, registry)
	if err != nil {
		return nil
	}

	result := make([]Entry, 0)

	if len(filters) == 0 {
		result = append(result, entries...)
		return result
	}

	for _, filter := range filters {
		for _, entry := range entries {
			if filter.Matches(entry) {
				result = append(result, entry)
			}
		}
	}
	return result
}

type MockGlobalState struct{}

func (m *MockGlobalState) GetCachedEntries(amcacheTable AmcacheTable, filters []filters.Filter) []Entry {
	return GetFilteredEntries(amcacheTable, filters)
}

func TestTables(t *testing.T) {
	mockState := &MockGlobalState{}
	cases := []struct {
		name         string
		amcacheTable AmcacheTable
	}{
		{"amcache_application", ApplicationTable},
		{"amcache_application_file", ApplicationFileTable},
		{"amcache_application_shortcut", ApplicationShortcutTable},
		{"amcache_driver_binary", DriverBinaryTable},
		{"amcache_device_pnp", DevicePnpTable},
	}

	for _, tc := range cases {
		log := logger.New(os.Stderr, false)
		t.Run(tc.name, func(t *testing.T) {
			generateFunc := tc.amcacheTable.GenerateFunc(mockState, log)
			rows, err := generateFunc(context.Background(), table.QueryContext{})
			if err != nil {
				t.Fatalf("Error generating rows for %s: %v", tc.name, err)
			}
			if len(rows) == 0 {
				t.Fatalf("No rows returned for %s", tc.name)
			}
			for _, row := range rows {
				for _, column := range tc.amcacheTable.Columns() {
					if _, ok := row[column.Name]; !ok {
						t.Errorf("Expected column %s in row, but not found", column.Name)
					}
				}
			}
		})
	}
}

func TestFiltering(t *testing.T) {

	mockState := &MockGlobalState{}
	cases := []struct {
		name         string
		amcacheTable AmcacheTable
		filterColumn string
	}{
		{"amcache_application", ApplicationTable, "program_id"},
		{"amcache_application_file", ApplicationFileTable, "file_id"},
		{"amcache_application_shortcut", ApplicationShortcutTable, "shortcut_program_id"},
		{"amcache_driver_binary", DriverBinaryTable, "driver_id"},
		{"amcache_device_pnp", DevicePnpTable, "driver_id"},
	}

	getQueryContext := func(filterColumn string, value string) table.QueryContext {
		constraint := table.Constraint{
			Operator:  table.OperatorEquals,
			Expression: value,
		}
		constraintList := table.ConstraintList{
			Affinity:    table.ColumnTypeText,
			Constraints: []table.Constraint{constraint},
		}
		return table.QueryContext{
			Constraints: map[string]table.ConstraintList{
				filterColumn: constraintList,
			},
		}
	}

	for _, tc := range cases {
		log := logger.New(os.Stderr, false)
		t.Run("Filtering "+tc.name, func(t *testing.T) {
			generateFunc := tc.amcacheTable.GenerateFunc(mockState, log)
			rows, err := generateFunc(context.Background(), table.QueryContext{})
			if err != nil {
				t.Fatalf("Error generating rows: %v", err)
			}
			if len(rows) == 0 {
				t.Fatalf("No rows returned for %s", tc.name)
			}
			filtered_rows, err := generateFunc(context.Background(), getQueryContext(tc.filterColumn, rows[0][tc.filterColumn]))
			if err != nil {
				t.Fatalf("Error generating filtered rows: %v", err)
			}
			if len(filtered_rows) == 0 {
				t.Fatalf("No filtered rows returned for %s", tc.name)
			}
			for _, row := range filtered_rows {
				if row[tc.filterColumn] != rows[0][tc.filterColumn] {
					t.Errorf("Filtering failed, expected %s=%s but got %s", tc.filterColumn, rows[0][tc.filterColumn], row[tc.filterColumn])
				}
			}
			if len(filtered_rows) >= len(rows) {
				t.Errorf("Filtering did not reduce the number of rows for %s", tc.name)
			}
		})
	}
}
