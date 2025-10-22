// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
	"context"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/encoding"
	"github.com/osquery/osquery-go/plugin/table"
)

// ApplicationShortcutEntry represents a single entry in the amcache application shortcut table.
// located at Root\\InventoryApplicationShortcut
type ApplicationShortcutEntry struct {
	LastWriteTime      int64  `osquery:"last_write_time"`
	ShortcutPath       string `osquery:"shortcut_path"`
	ShortcutTargetPath string `osquery:"shortcut_target_path"`
	ShortcutAumid      string `osquery:"shortcut_aumid"`
	ShortcutProgramId  string `osquery:"shortcut_program_id"`
}

// FilterValue returns the index value for the ApplicationShortcutEntry, which is the ShortcutProgramId.
func (ase *ApplicationShortcutEntry) FilterValue() string {
	return ase.ShortcutProgramId
}

// ToMap converts the ApplicationShortcutEntry to a map[string]string representation.
func (ase *ApplicationShortcutEntry) ToMap() (map[string]string, error) {
	mapped, err := encoding.MarshalToMap(ase)
	return mapped, err
}

// ApplicationShortcutTable implements the TableInterface for the amcache application shortcut table.
type ApplicationShortcutTable struct{}

// Type returns the TableType for the ApplicationShortcutTable.
func (ast *ApplicationShortcutTable) Type() TableType {
	return ApplicationShortcutTableType
}

// FilterColumn returns the name of the column used for filtering entries in the ApplicationShortcutTable.
func (ast *ApplicationShortcutTable) FilterColumn() string {
	return "shortcut_program_id"
}

// Columns returns the column definitions for the ApplicationShortcutTable.
func (ast *ApplicationShortcutTable) Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.BigIntColumn("last_write_time"),
		table.TextColumn("shortcut_path"),
		table.TextColumn("shortcut_target_path"),
		table.TextColumn("shortcut_aumid"),
		table.TextColumn("shortcut_program_id"),
	}
}

// GenerateFunc generates the data for the ApplicationShortcutTable based on the provided GlobalStateInterface.
func (ast *ApplicationShortcutTable) GenerateFunc(state GlobalStateInterface) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		programIds := GetConstraintsFromQueryContext(ast.FilterColumn(), queryContext)
		entries := state.GetCachedEntries(ast.Type(), programIds...)

		rows := make([]map[string]string, 0, len(entries))
		for _, entry := range entries {
			mapped, err := entry.ToMap()
			if err != nil {
				return nil, err
			}
			rows = append(rows, mapped)
		}
		return rows, nil
	}
}