// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
	"context"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/registry"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/testdata"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/filters"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	"github.com/osquery/osquery-go/plugin/table"
	"os"
	"testing"
)

type MockGlobalState struct{}

func (m *MockGlobalState) GetCachedEntries(amcacheTable AmcacheTable, filters []filters.Filter, log *logger.Logger) ([]Entry, error) {
	registry, _, err := registry.LoadRegistry(testdata.GetTestHivePathOrFatal(nil), log)
	if err != nil {
		return nil, err
	}
	entries, err := GetEntriesFromRegistry(amcacheTable, registry, log)
	if err != nil {
		return nil, err
	}

	result := make([]Entry, 0)

	if len(filters) == 0 {
		result = append(result, entries...)
		return result, nil
	}

	for _, filter := range filters {
		for _, entry := range entries {
			if filter.Matches(entry) {
				result = append(result, entry)
			}
		}
	}
	return result, nil
}

func TestTables(t *testing.T) {
	log := logger.New(os.Stdout, true)
	mockState := &MockGlobalState{}
	amcacheTable := *GetAmcacheTableByName(TableNameApplication)
	generateFunc := amcacheTable.GenerateFunc(mockState, log)

	rows, err := generateFunc(context.Background(), table.QueryContext{})
	if err != nil {
		t.Fatalf("Error generating rows for %s: %v", amcacheTable.Name, err)
	}
	if len(rows) == 0 {
		t.Fatalf("No rows returned for %s", amcacheTable.Name)
	}

	queryContext := table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"name": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: rows[0]["name"],
					},
				},
			},
		},
	}
	filteredRows, err := generateFunc(context.Background(), queryContext)
	if err != nil {
		t.Fatalf("Error generating filtered rows for %s: %v", amcacheTable.Name, err)
	}
	if len(filteredRows) == 0 {
		t.Fatalf("No filtered rows returned for %s", amcacheTable.Name)
	}
	if len(filteredRows) >= len(rows) {
		t.Fatalf("Expected less than %d filtered rows, got %d", len(rows), len(filteredRows))
	}
}
