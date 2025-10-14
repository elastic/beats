// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package main

import (
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/state"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables/application"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables/application_file"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables/application_shortcut"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables/device_pnp"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables/driver_binary"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

func RegisterTables(server *osquery.ExtensionManagerServer) {

	globalState := state.GetInstance()

	server.RegisterPlugin(table.NewPlugin("amcache_application", application.Columns(), application.GenerateFunc(globalState)))
	server.RegisterPlugin(table.NewPlugin("amcache_application_file", application_file.Columns(), application_file.GenerateFunc(globalState)))
	server.RegisterPlugin(table.NewPlugin("amcache_application_shortcut", application_shortcut.Columns(), application_shortcut.GenerateFunc(globalState)))
	server.RegisterPlugin(table.NewPlugin("amcache_device_pnp", device_pnp.Columns(), device_pnp.GenerateFunc(globalState)))
	server.RegisterPlugin(table.NewPlugin("amcache_driver_binary", driver_binary.Columns(), driver_binary.GenerateFunc(globalState)))
}
