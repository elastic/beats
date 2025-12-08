// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package main

import (
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/browserhistory"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/hooks"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

func RegisterAmcacheTables(server *osquery.ExtensionManagerServer, log *logger.Logger, hooks *hooks.HookManager) {
	amcacheGlobalState := tables.GetAmcacheState()
	for _, t := range tables.AllAmcacheTables() {
		server.RegisterPlugin(table.NewPlugin(string(t.Name), t.Columns(), t.GenerateFunc(amcacheGlobalState, log)))
	}
	amcache.RegisterHooks(hooks)
}

func RegisterTables(server *osquery.ExtensionManagerServer, log *logger.Logger, hooks *hooks.HookManager) {
	server.RegisterPlugin(table.NewPlugin("elastic_browser_history", browserhistory.GetColumns(), browserhistory.GetGenerateFunc(log)))
	RegisterAmcacheTables(server, log, hooks)
}
