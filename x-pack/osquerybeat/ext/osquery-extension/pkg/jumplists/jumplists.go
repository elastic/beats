// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package jumplists

import (
	"context"

	"github.com/osquery/osquery-go/plugin/table"
)

func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("test_column"),
	}
}

func GenerateFunc() table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	rows := make([]map[string]string, 0, 1)
	row := map[string]string{
		"test_column": "test_value",
	}
	rows = append(rows, row)
	return rows, nil
	}
}