// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fileanalysis

import (
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/encoding"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tablespec"
)

func TableSpec() *tablespec.TableSpec {
	columns, err := encoding.GenerateColumnDefinitions(fileAnalysis{})
	if err != nil {
		panic(err)
	}
	return tablespec.NewTableSpec(
		"elastic_file_analysis",
		"File analysis table for macOS files, providing metadata and content insights",
		[]string{"darwin"},
		columns,
		generate,
	)
}
