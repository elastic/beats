// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
	"context"
	"os"
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/registry"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/filters"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

type MockGlobalState struct{}

func (m *MockGlobalState) GetCachedEntries(amcacheTable AmcacheTable, filters []filters.Filter, log *logger.Logger) ([]Entry, error) {
	registry, _, err := registry.LoadRegistry("../testdata/Amcache.hve", log)
	if err != nil {
		return nil, err
	}
	entries, err := GetEntriesFromRegistry(amcacheTable, registry, log)
	if err != nil {
		return nil, err
	}

	if len(filters) == 0 {
		return entries, nil
	}

	filteredEntries := make([]Entry, 0)
	for _, filter := range filters {
		for _, entry := range entries {
			if filter.Matches(entry) {
				filteredEntries = append(filteredEntries, entry)
			}
		}
	}
	return filteredEntries, nil
}

func TestTables(t *testing.T) {
	log := logger.New(os.Stdout, true)
	mockState := &MockGlobalState{}
	amcacheTable := *GetAmcacheTableByName(TableNameApplication)
	generateFunc := amcacheTable.GenerateFunc(mockState, log)

	rows, err := generateFunc(context.Background(), table.QueryContext{})
	assert.NoError(t, err, "failed to generate rows for %s", amcacheTable.Name)
	assert.NotEmpty(t, rows, "no rows returned for %s", amcacheTable.Name)

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
	assert.NoError(t, err, "failed to generate filtered rows for %s", amcacheTable.Name)
	assert.NotEmpty(t, filteredRows, "no filtered rows returned for %s", amcacheTable.Name)
	assert.Less(t, len(filteredRows), len(rows), "expected less than %d filtered rows, got %d", len(rows), len(filteredRows))
}
