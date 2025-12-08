// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package main

import (
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/browserhistory"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/hooks"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables"
)

func RegisterTables(server *osquery.ExtensionManagerServer, log *logger.Logger, postHooks *hooks.HookManager) {
	server.RegisterPlugin(table.NewPlugin("host_users", tables.HostUsersColumns(), tables.GetHostUsersGenerateFunc(log)))
	server.RegisterPlugin(table.NewPlugin("host_groups", tables.HostGroupsColumns(), tables.GetHostGroupsGenerateFunc(log)))
	server.RegisterPlugin(table.NewPlugin("host_processes", tables.HostProcessesColumns(), tables.GetHostProcessesGenerateFunc(log)))
	server.RegisterPlugin(table.NewPlugin("elastic_browser_history", browserhistory.GetColumns(), browserhistory.GetGenerateFunc(log)))
}
