// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tables

import (
	"github.com/osquery/osquery-go"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated"
)

// RegisterTables registers all generated tables with the osquery extension server.
// This is the stable entry point that wraps the generated registry.
func RegisterTables(server *osquery.ExtensionManagerServer, log *logger.Logger) {
	generated.RegisterTables(server, log)
}
