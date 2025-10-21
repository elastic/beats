// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package main

import (
	"log"
	"os"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/state"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

func RegisterTables(server *osquery.ExtensionManagerServer) {

	globalState := state.GetGlobalState()
	server.RegisterPlugin(table.NewPlugin("amcache_application", tables.ApplicationColumns(), tables.ApplicationGenerateFunc(globalState)))
	server.RegisterPlugin(table.NewPlugin("amcache_application_file", tables.ApplicationFileColumns(), tables.ApplicationFileGenerateFunc(globalState)))
	server.RegisterPlugin(table.NewPlugin("amcache_application_shortcut", tables.ApplicationShortcutColumns(), tables.ApplicationShortcutGenerateFunc(globalState)))
	server.RegisterPlugin(table.NewPlugin("amcache_device_pnp", tables.DevicePnpColumns(), tables.DevicePnpGenerateFunc(globalState)))
	server.RegisterPlugin(table.NewPlugin("amcache_driver_binary", tables.DriverBinaryColumns(), tables.DriverBinaryGenerateFunc(globalState)))
}

func CreateViews(socket *string) {
	filepath := "C:\\app.log"
	if _, err := os.Stat(filepath); err == nil {
		os.Remove(filepath)
	}
	file, err := os.OpenFile("C:\\app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err) // Use standard logger (stderr) to report this fatal error
	}
	defer file.Close()
	log.SetOutput(file)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("Starting AMCache view creation")
	amcacheView := View{
		requiredTables: []string{"amcache_application", "amcache_application_file"},
		createViewQuery: "CREATE VIEW amcache_applications as SELECT * FROM amcache_application_file JOIN amcache_application using (program_id);",
	}
	amcacheView.CreateView(socket)
	log.Println("Finished AMCache view creation")
}
