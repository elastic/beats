// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
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

	entries, err := mockState.GetCachedEntries(amcacheTable, nil, log)
	assert.NoError(t, err, "failed to get entries for %s", amcacheTable.Name)
	if len(entries) == 0 {
		t.Skip("no amcache testdata (e.g. Amcache.hve); run on Windows with testdata to exercise filtering")
	}

	// Test filtering: filter by name from first entry
	appEntry := entries[0].(*ApplicationEntry)
	nameFilter := appEntry.Name
	fltrs := []filters.Filter{
		{ColumnName: "name", Operator: table.OperatorEquals, Expression: nameFilter},
	}
	filteredEntries, err := mockState.GetCachedEntries(amcacheTable, fltrs, log)
	assert.NoError(t, err, "failed to get filtered entries for %s", amcacheTable.Name)
	assert.NotEmpty(t, filteredEntries, "no filtered entries returned for %s", amcacheTable.Name)
	assert.LessOrEqual(t, len(filteredEntries), len(entries), "filtered count should be <= unfiltered")
}
