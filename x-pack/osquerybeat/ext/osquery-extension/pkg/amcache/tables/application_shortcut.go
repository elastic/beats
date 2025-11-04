// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

// ApplicationShortcutEntry represents a single entry in the amcache application shortcut table.
// located at Root\\InventoryApplicationShortcut
type ApplicationShortcutEntry struct {
	BaseEntry
	ShortcutPath       string `osquery:"shortcut_path"`
	ShortcutTargetPath string `osquery:"shortcut_target_path"`
	ShortcutAumid      string `osquery:"shortcut_aumid"`
	ShortcutProgramId  string `osquery:"shortcut_program_id"`
}

func (e *ApplicationShortcutEntry) PostProcess() {
}
