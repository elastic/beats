// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
	"context"
	"fmt"
	"time"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/encoding"
	"github.com/osquery/osquery-go/plugin/table"
)

// ApplicationFileEntry represents a single entry in the amcache application file table.
// located at Root\\InventoryApplicationFile
type JumpListEntry struct {
	LinkPath         string  `osquery:"link_path"`
	TargetCreatedTime time.Time `osquery:"target_created_time" format:"unix"`
	TargetModifiedTime time.Time `osquery:"target_modified_time"`
	TargetAccessedTime time.Time `osquery:"target_accessed_time"`
	TargetSize         int64  `osquery:"target_size"`
	TargetPath         string `osquery:"target_path"`
}

// JumpListTable implements the TableInterface for the jumplists table.
type JumpListTable struct{}

// Columns returns the osquery column definitions for the jumplists table
func (jlt *JumpListTable) Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("link_path"),
		table.BigIntColumn("target_created_time"),
		table.BigIntColumn("target_modified_time"),
		table.BigIntColumn("target_accessed_time"),
		table.BigIntColumn("target_size"),
		table.TextColumn("target_path"),
	}
}

func (jle *JumpListEntry) ToMap() (map[string]string, error) {
	return encoding.MarshalToMap(jle)
}
	// GenerateFunc generates the data for the ApplicationFileTable based on the provided GlobalStateInterface.
func (jlt *JumpListTable) GenerateFunc() table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		rows := make([]map[string]string, 0)

		jle := JumpListEntry{
			LinkPath: "test_value",
			TargetCreatedTime: time.Now(),
			TargetModifiedTime: time.Now(),
			TargetAccessedTime: time.Now(),
			TargetSize: 100,
			TargetPath: "test_value",
		}
		row, err := jle.ToMap()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal jump list entry: %w", err)
		}
		rows = append(rows, row)
		return rows, nil
	}
}
