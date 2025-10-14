// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package application_shortcut

import (
	"context"
	"fmt"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/interfaces"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/utilities"
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

func Columns() []table.ColumnDefinition {
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

func GetApplicationShortcutEntriesFromRegistry(registry *regparser.Registry) (map[string][]interfaces.Entry, error) {
	if registry == nil {
		return nil, fmt.Errorf("registry is nil")
	}

	keyName := "Root\\InventoryApplicationShortcut"
	keyNode := registry.OpenKey(keyName)
	if keyNode == nil {
		return nil, fmt.Errorf("error opening key: %s", keyName)
	}

	applicationEntries := make(map[string][]interfaces.Entry, len(keyNode.Subkeys()))
	for _, subkey := range keyNode.Subkeys() {
		ase := &ApplicationShortcutEntry{}
		interfaces.FillInEntryFromKey(ase, subkey)

		applicationEntries[ase.ShortcutProgramId] = append(applicationEntries[ase.ShortcutProgramId], ase)
	}
	return applicationEntries, nil
}

func GenerateFunc(state interfaces.GlobalState) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		programIds := utilities.GetConstraintsFromQueryContext("shortcut_program_id", queryContext)
		rows := state.GetApplicationShortcutEntries(programIds...)
		return interfaces.RowsAsStringMapArray(rows), nil
	}
}
