// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package main

import (
	"context"

	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/client"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/hooks"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"

	// macadmins extension tables (cross-platform and linux-compatible)
	"github.com/macadmins/osquery-extension/tables/chromeuserprofiles"
	"github.com/macadmins/osquery-extension/tables/crowdstrike_falcon"
	"github.com/macadmins/osquery-extension/tables/fileline"
	"github.com/macadmins/osquery-extension/tables/puppet"
)

func RegisterTables(server *osquery.ExtensionManagerServer, log *logger.Logger, _ *hooks.HookManager, _ *client.ResilientClient) {
	socketPath := *socket

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

	// Linux/Darwin table
	server.RegisterPlugin(table.NewPlugin(
		"crowdstrike_falcon",
		crowdstrike_falcon.CrowdstrikeFalconColumns(),
		func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
			return crowdstrike_falcon.CrowdstrikeFalconGenerate(ctx, queryContext, socketPath)
		}))
	log.Infof("Registered macadmins table: crowdstrike_falcon")
}
