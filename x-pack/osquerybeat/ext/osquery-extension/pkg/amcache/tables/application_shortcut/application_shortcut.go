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

type ApplicationShortcutTable struct {
	Entries []interfaces.Entry
}

func (t *ApplicationShortcutTable) AddRow(key *regparser.CM_KEY_NODE) error {
	ae := &ApplicationShortcutEntry{}
	interfaces.FillInEntryFromKey(ae, key)
	t.Entries = append(t.Entries, ae)
	return nil
}

func (t *ApplicationShortcutTable) Rows() []interfaces.Entry {
	return t.Entries
}

func (t *ApplicationShortcutTable) KeyName() string {
	return "Root\\InventoryApplicationShortcut"
}

func GenerateFunc(hiveReader *utilities.HiveReader) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		table := ApplicationShortcutTable{}
		err := interfaces.BuildTableFromRegistry(&table, hiveReader, ctx, queryContext)
		if err != nil {
			return nil, fmt.Errorf("failed to build ApplicationShortcutTable: %w", err)
		}
		return interfaces.RowsAsStringMapArray(&table), nil
	}
}
