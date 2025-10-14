// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package main

import (
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables/application"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables/application_file"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables/application_shortcut"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables/device_pnp"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables/driver_binary"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/utilities"
)

func RegisterTables(server *osquery.ExtensionManagerServer) {
	hiveReader := utilities.HiveReader{FilePath: "C:\\Windows\\AppCompat\\Programs\\Amcache.hve"}

	server.RegisterPlugin(table.NewPlugin("amcache_application", application.ApplicationColumns(), application.GenerateFunc(&hiveReader)))
	server.RegisterPlugin(table.NewPlugin("amcache_application_file", application_file.ApplicationFileColumns(), application_file.GenerateFunc(&hiveReader)))
	server.RegisterPlugin(table.NewPlugin("amcache_application_shortcut", application_shortcut.ApplicationShortcutColumns(), application_shortcut.GenerateFunc(&hiveReader)))
	server.RegisterPlugin(table.NewPlugin("amcache_device_pnp", device_pnp.DevicePnpColumns(), device_pnp.GenerateFunc(&hiveReader)))
	server.RegisterPlugin(table.NewPlugin("amcache_driver_binary", driver_binary.DriverBinaryColumns(), driver_binary.GenerateFunc(&hiveReader)))
}
