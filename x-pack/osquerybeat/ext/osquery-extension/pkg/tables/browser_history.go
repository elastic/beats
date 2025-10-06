// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tables

import (
	"context"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/browserhistory"
)

func BrowserHistoryColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.BigIntColumn("timestamp"),
		table.TextColumn("url"),
		table.TextColumn("title"),
		table.TextColumn("browser"),
		table.TextColumn("transition_type"),
		table.TextColumn("referring_url"),
		table.BigIntColumn("visit_chain_id"),
		table.BigIntColumn("prior_visit_chain_id"),
		table.BigIntColumn("visit_duration_ms"),
		table.IntegerColumn("typed_count"),
		table.TextColumn("source_path"),
	}
}

func GetBrowserHistoryGenerateFunc(log func(m string, kvs ...any)) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		return browserhistory.GetTable(ctx, queryContext, log)
	}
}
