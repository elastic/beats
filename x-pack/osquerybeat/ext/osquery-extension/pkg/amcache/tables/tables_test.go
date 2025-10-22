// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
	"context"
	"testing"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/testdata"
	"github.com/osquery/osquery-go/plugin/table"
)

func GetFilteredEntries(tableType TableType, ids ...string) []Entry {
	registry, err := LoadRegistry(testdata.GetTestHivePathOrFatal(nil))
	if err != nil {
		return nil
	}
	entriesMap, err := GetEntriesFromRegistry(tableType, registry)
	if err != nil {
		return nil
	}

	result := make([]Entry, 0)

	if len(ids) == 0 {
		for _, entries := range entriesMap {
			for _, entry := range entries {
				result = append(result, entry)
			}
		}
		return result
	}

	for _, id := range ids {
		if entries, ok := entriesMap[id]; ok {
			for _, entry := range entries {
				result = append(result, entry)
			}
		}
	}
	return result
}

type MockGlobalState struct{}

func (m *MockGlobalState) GetCachedEntries(tableType TableType, ids ...string) []Entry {
	return GetFilteredEntries(tableType, ids...)
}

func TestTables(t *testing.T) {
	mockState := &MockGlobalState{}
	cases := []struct {
		name         string
		table        TableInterface
	}{
		{"amcache_application", &ApplicationTable{}},
		{"amcache_application_file", &ApplicationFileTable{}},
		{"amcache_application_shortcut", &ApplicationShortcutTable{}},
		{"amcache_driver_binary", &DriverBinaryTable{}},
		{"amcache_device_pnp", &DevicePnpTable{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			generateFunc := tc.table.GenerateFunc(mockState)
			rows, err := generateFunc(context.Background(), table.QueryContext{})
			if err != nil {
				t.Fatalf("Error generating rows for %s: %v", tc.name, err)
			}
			if len(rows) == 0 {
				t.Fatalf("No rows returned for %s", tc.name)
			}
			for _, row := range rows {
				for _, column := range tc.table.Columns() {
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
		table        TableInterface
	}{
		{"amcache_application", &ApplicationTable{}},
		{"amcache_application_file", &ApplicationFileTable{}},
		{"amcache_application_shortcut", &ApplicationShortcutTable{}},
		{"amcache_driver_binary", &DriverBinaryTable{}},
		{"amcache_device_pnp", &DevicePnpTable{}},
	}

	getQueryContext := func(field string, value string) table.QueryContext {
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
				field: constraintList,
			},
		}
	}

	for _, tc := range cases {
		t.Run("Filtering "+tc.name, func(t *testing.T) {
			generateFunc := tc.table.GenerateFunc(mockState)
			rows, err := generateFunc(context.Background(), table.QueryContext{})
			if err != nil {
				t.Fatalf("Error generating rows: %v", err)
			}
			if len(rows) == 0 {
				t.Fatalf("No rows returned for %s", tc.name)
			}
			filtered_rows, err := generateFunc(context.Background(), getQueryContext(tc.table.FilterColumn(), rows[0][tc.table.FilterColumn()]))
			if err != nil {
				t.Fatalf("Error generating filtered rows: %v", err)
			}
			if len(filtered_rows) == 0 {
				t.Fatalf("No filtered rows returned for %s", tc.name)
			}
			for _, row := range filtered_rows {
				if row[tc.table.FilterColumn()] != rows[0][tc.table.FilterColumn()] {
					t.Errorf("Filtering failed, expected %s=%s but got %s", tc.table.FilterColumn(), rows[0][tc.table.FilterColumn()], row[tc.table.FilterColumn()])
				}
			}
			if len(filtered_rows) >= len(rows) {
				t.Errorf("Filtering did not reduce the number of rows for %s", tc.name)
			}
		})
	}
}
