// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package main

import (
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/state"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

func RegisterTables(server *osquery.ExtensionManagerServer) {

	globalState := state.GetGlobalState()
	applicationTable := &tables.ApplicationTable{}
	applicationFileTable := &tables.ApplicationFileTable{}
	applicationShortcutTable := &tables.ApplicationShortcutTable{}
	devicePnpTable := &tables.DevicePnpTable{}
	driverBinaryTable := &tables.DriverBinaryTable{}

	server.RegisterPlugin(table.NewPlugin("amcache_application", applicationTable.Columns(), applicationTable.GenerateFunc(globalState)))
	server.RegisterPlugin(table.NewPlugin("amcache_application_file", applicationFileTable.Columns(), applicationFileTable.GenerateFunc(globalState)))
	server.RegisterPlugin(table.NewPlugin("amcache_application_shortcut", applicationShortcutTable.Columns(), applicationShortcutTable.GenerateFunc(globalState)))
	server.RegisterPlugin(table.NewPlugin("amcache_device_pnp", devicePnpTable.Columns(), devicePnpTable.GenerateFunc(globalState)))
	server.RegisterPlugin(table.NewPlugin("amcache_driver_binary", driverBinaryTable.Columns(), driverBinaryTable.GenerateFunc(globalState)))
}

func CreateViews(socket *string) {
	view := View{
		requiredTables: []string{"amcache_application", "amcache_application_file"},
		createViewQuery: "CREATE VIEW amcache_applications as SELECT * FROM amcache_application_file JOIN amcache_application using (program_id);",
	}
	view.CreateView(socket)
}
