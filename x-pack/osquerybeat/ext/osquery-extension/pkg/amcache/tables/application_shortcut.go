// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
	"context"
	"fmt"
	"github.com/osquery/osquery-go/plugin/table"
	"www.velocidex.com/golang/regparser"
)

type ApplicationShortcutEntry struct {
	LastWriteTime      int64  `json:"last_write_time,string"`
	ShortcutPath       string `json:"shortcut_path"`
	ShortcutTargetPath string `json:"shortcut_target_path"`
	ShortcutAumid      string `json:"shortcut_aumid"`
	ShortcutProgramId  string `json:"shortcut_program_id"`
}

func ApplicationShortcutColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.BigIntColumn("last_write_time"),
		table.TextColumn("shortcut_path"),
		table.TextColumn("shortcut_target_path"),
		table.TextColumn("shortcut_aumid"),
		table.TextColumn("shortcut_program_id"),
	}
}

func (ae *ApplicationShortcutEntry) FieldMappings() map[string]*string {
	return map[string]*string{
		"ShortcutPath":       &ae.ShortcutPath,
		"ShortcutTargetPath": &ae.ShortcutTargetPath,
		"ShortcutAumid":      &ae.ShortcutAumid,
		"ShortcutProgramId":  &ae.ShortcutProgramId,
	}
}

func (ae *ApplicationShortcutEntry) SetLastWriteTime(t int64) {
	ae.LastWriteTime = t
}

func GetApplicationShortcutEntriesFromRegistry(registry *regparser.Registry) (map[string][]Entry, error) {
	if registry == nil {
		return nil, fmt.Errorf("registry is nil")
	}

	keyNode := registry.OpenKey(applicationShortcutKeyPath)
	if keyNode == nil {
		return nil, fmt.Errorf("error opening key: %s", applicationShortcutKeyPath)
	}

	applicationEntries := make(map[string][]Entry, len(keyNode.Subkeys()))
	for _, subkey := range keyNode.Subkeys() {
		ase := &ApplicationShortcutEntry{}
		FillInEntryFromKey(ase, subkey)

		applicationEntries[ase.ShortcutProgramId] = append(applicationEntries[ase.ShortcutProgramId], ase)
	}
	return applicationEntries, nil
}

func ApplicationShortcutGenerateFunc(state GlobalStateInterface) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		programIds := GetConstraintsFromQueryContext("shortcut_program_id", queryContext)
		rows := state.GetApplicationShortcutEntries(programIds...)
		return RowsAsStringMapArray(rows), nil
	}
}
