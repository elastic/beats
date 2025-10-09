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
		// Universal fields (available across all browsers)
		table.BigIntColumn("timestamp"),
		table.TextColumn("url"),
		table.TextColumn("title"),
		table.TextColumn("browser"),
		table.TextColumn("parser"),
		table.TextColumn("user"),
		table.TextColumn("profile_name"),
		table.TextColumn("transition_type"),
		table.TextColumn("referring_url"),
		table.BigIntColumn("visit_id"),
		table.BigIntColumn("from_visit_id"),
		table.BigIntColumn("url_id"),
		table.BigIntColumn("visit_count"),
		table.IntegerColumn("typed_count"),
		table.TextColumn("visit_source"),
		table.IntegerColumn("is_hidden"),
		table.TextColumn("source_path"),

		// Chromium-specific fields (Chrome, Edge, Brave, etc.)
		table.BigIntColumn("ch_visit_duration_ms"), // Only available in Chromium-based browsers

		// Firefox-specific fields
		table.BigIntColumn("ff_session_id"), // Firefox session tracking
		table.BigIntColumn("ff_frecency"),   // Firefox user interest algorithm

		// Safari-specific fields
		table.TextColumn("sf_domain_expansion"),   // Safari domain classification
		table.IntegerColumn("sf_load_successful"), // Whether page loaded successfully
	}
}

func GetBrowserHistoryGenerateFunc(log func(m string, kvs ...any)) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		return browserhistory.GetTableRows(ctx, queryContext, log)
	}
}
