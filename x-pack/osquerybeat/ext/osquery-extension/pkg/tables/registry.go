// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tables

import (
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
	
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

var registry []TableSpec

// RegisterTableSpec registers a table spec.
// This is called from each generated table's init() function.
func RegisterTableSpec(spec TableSpec) {
	registry = append(registry, spec)
}

// TableSpec contains metadata and references for a generated table.
type TableSpec struct {
	Name         string
	Description  string
	Platforms    []string
	TableName    string
	Columns      func() []table.ColumnDefinition
	GenerateFunc func(*logger.Logger) (table.GenerateFunc, error)
}

// RegisterTables registers all tables in the registry with the osquery extension server.
func RegisterTables(server *osquery.ExtensionManagerServer, log *logger.Logger) {
	for _, spec := range registry {
		genFunc, err := spec.GenerateFunc(log)
		if err != nil {
			log.Errorf("Failed to get generate function for table %s: %v", spec.Name, err)
			continue
		}
		server.RegisterPlugin(table.NewPlugin(spec.TableName, spec.Columns(), genFunc))
		log.Infof("Registered table: %s", spec.TableName)
	}
}
