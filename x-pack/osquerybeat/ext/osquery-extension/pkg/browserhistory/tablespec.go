// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"context"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/encoding"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tablespec"
)

func TableSpec() *tablespec.TableSpec {
	columns, err := encoding.GenerateColumnDefinitions(visit{})
	if err != nil {
		panic(err)
	}
	return tablespec.NewTableSpec(
		"elastic_browser_history",
		"Cross-platform browser history analysis table supporting Chrome, Firefox, Safari, Edge, and Brave browsers",
		[]string{"linux", "darwin", "windows"},
		columns,
		generate,
	)
}

func generate(log *logger.Logger) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		return getTableRows(ctx, queryContext, log)
	}
}
