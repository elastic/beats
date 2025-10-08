// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
	"context"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache"
)

// AmcacheColumns returns the column definitions for the Amcache osquery table.
func AmcacheColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("name"),
		table.BigIntColumn("first_run_time"),
		table.TextColumn("program_id"),
		table.TextColumn("file_id"),
		table.TextColumn("lower_case_long_path"),
		table.TextColumn("original_file_name"),
		table.TextColumn("publisher"),
		table.TextColumn("version"),
		table.TextColumn("bin_file_version"),
		table.TextColumn("binary_type"),
		table.TextColumn("product_name"),
		table.TextColumn("product_version"),
		table.TextColumn("link_date"),
		table.TextColumn("bin_product_version"),
		table.BigIntColumn("size"),
		table.BigIntColumn("language"),
		table.BigIntColumn("usn"),
	}
}

// GetAmcacheGenerateFunc returns the generate function for the Amcache osquery table.
func GetAmcacheGenerateFunc() table.GenerateFunc {
	// TODO: Determine if want to add caching if the query ends up being too slow.  
	// Currently reads live hive each time. If we decide to cache, the GenAmcacheTable function
	// already supports passing a file path to read from.
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		return amcache.GenAmcacheTable(ctx, queryContext)
	}
}
