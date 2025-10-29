// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
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

// Columns returns the column definitions for the ApplicationShortcutTable.
func ApplicationShortcutColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.BigIntColumn("last_write_time"),
		table.TextColumn("shortcut_path"),
		table.TextColumn("shortcut_target_path"),
		table.TextColumn("shortcut_aumid"),
		table.TextColumn("shortcut_program_id"),
	}
}