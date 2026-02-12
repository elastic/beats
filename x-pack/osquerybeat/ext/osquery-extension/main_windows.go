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
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/client"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/hooks"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"

	// macadmins extension tables (cross-platform)
	"github.com/macadmins/osquery-extension/tables/chromeuserprofiles"
	"github.com/macadmins/osquery-extension/tables/fileline"
	"github.com/macadmins/osquery-extension/tables/puppet"
)

func RegisterAmcacheTables(server *osquery.ExtensionManagerServer, log *logger.Logger, hooks *hooks.HookManager) {
	amcacheGlobalState := tables.GetAmcacheState()
	for _, t := range tables.AllAmcacheTables() {
		server.RegisterPlugin(table.NewPlugin(string(t.Name), t.Columns(), t.GenerateFunc(amcacheGlobalState, log)))
	}
	amcache.RegisterHooks(hooks)
}

func RegisterTables(server *osquery.ExtensionManagerServer, log *logger.Logger, hooks *hooks.HookManager, client *client.ResilientClient) {
	server.RegisterPlugin(table.NewPlugin("elastic_jumplists", jumplists.GetColumns(), jumplists.GetGenerateFunc(log, client)))
	RegisterAmcacheTables(server, log, hooks)

	// Cross-platform tables from macadmins extension
	server.RegisterPlugin(table.NewPlugin("puppet_info", puppet.PuppetInfoColumns(), puppet.PuppetInfoGenerate))
	log.Infof("Registered macadmins table: puppet_info")

	server.RegisterPlugin(table.NewPlugin("puppet_logs", puppet.PuppetLogsColumns(), puppet.PuppetLogsGenerate))
	log.Infof("Registered macadmins table: puppet_logs")

	server.RegisterPlugin(table.NewPlugin("puppet_state", puppet.PuppetStateColumns(), puppet.PuppetStateGenerate))
	log.Infof("Registered macadmins table: puppet_state")

	server.RegisterPlugin(table.NewPlugin("puppet_facts", puppet.PuppetFactsColumns(), puppet.PuppetFactsGenerate))
	log.Infof("Registered macadmins table: puppet_facts")

	server.RegisterPlugin(table.NewPlugin("google_chrome_profiles", chromeuserprofiles.GoogleChromeProfilesColumns(), chromeuserprofiles.GoogleChromeProfilesGenerate))
	log.Infof("Registered macadmins table: google_chrome_profiles")

	server.RegisterPlugin(table.NewPlugin("file_lines", fileline.FileLineColumns(), fileline.FileLineGenerate))
	log.Infof("Registered macadmins table: file_lines")
}
