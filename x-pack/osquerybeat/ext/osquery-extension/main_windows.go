// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package main

import (
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/state"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/browserhistory"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/views"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

func RegisterAmcacheTables(server *osquery.ExtensionManagerServer, log *logger.Logger) {
	amcacheGlobalState := state.GetAmcacheGlobalState()
	for _, t := range tables.AllAmcacheTables() {
		server.RegisterPlugin(table.NewPlugin(string(t.Name), t.Columns(), t.GenerateFunc(amcacheGlobalState, log)))
	}
}

func RegisterTables(server *osquery.ExtensionManagerServer, log *logger.Logger) {
	server.RegisterPlugin(table.NewPlugin("browser_history", browserhistory.GetColumns(), browserhistory.GetGenerateFunc(log)))
	RegisterAmcacheTables(server, log)
}

func CreateViews(socket *string, log *logger.Logger) {
	view := views.NewView(
		"V_AmcacheApplications",
		[]string{"amcache_application", "amcache_application_file"},
		`CREATE VIEW V_AmcacheApplications AS
		SELECT
			app.*,
			file.*
		FROM
			amcache_application AS app
		LEFT JOIN amcache_application_file AS file ON app.program_id = file.program_id
		UNION
		SELECT
			app.*,
			file.*
		FROM
			amcache_application_file AS file
		LEFT JOIN amcache_application AS app ON file.program_id = app.program_id
		WHERE
			app.program_id IS NULL;`)
	err := views.CreateView(socket, view, log)
	if err != nil {
		log.Fatalf("Error creating view %s: %v\n", view.Name(), err)
	}
}
