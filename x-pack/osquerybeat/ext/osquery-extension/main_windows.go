// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package main

import (
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables"
)

func RegisterTables(server *osquery.ExtensionManagerServer) {
	server.RegisterPlugin(table.NewPlugin("amcache", tables.AmcacheColumns(), tables.GetAmcacheGenerateFunc()))
}
