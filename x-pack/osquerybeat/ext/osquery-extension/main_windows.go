// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package main

import (
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/state"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/views"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

func RegisterAmcacheTables(server *osquery.ExtensionManagerServer) {
	amcacheGlobalState := state.GetAmcacheGlobalState()
	applicationTable := &tables.ApplicationTable{}
	applicationFileTable := &tables.ApplicationFileTable{}
	applicationShortcutTable := &tables.ApplicationShortcutTable{}
	devicePnpTable := &tables.DevicePnpTable{}
	driverBinaryTable := &tables.DriverBinaryTable{}

	server.RegisterPlugin(table.NewPlugin("amcache_application", applicationTable.Columns(), applicationTable.GenerateFunc(amcacheGlobalState)))
	server.RegisterPlugin(table.NewPlugin("amcache_application_file", applicationFileTable.Columns(), applicationFileTable.GenerateFunc(amcacheGlobalState)))
	server.RegisterPlugin(table.NewPlugin("amcache_application_shortcut", applicationShortcutTable.Columns(), applicationShortcutTable.GenerateFunc(amcacheGlobalState)))
	server.RegisterPlugin(table.NewPlugin("amcache_device_pnp", devicePnpTable.Columns(), devicePnpTable.GenerateFunc(amcacheGlobalState)))
	server.RegisterPlugin(table.NewPlugin("amcache_driver_binary", driverBinaryTable.Columns(), driverBinaryTable.GenerateFunc(amcacheGlobalState)))
}

func RegisterTables(server *osquery.ExtensionManagerServer) {
	RegisterAmcacheTables(server)
}

func CreateViews(socket *string) {
	applicationsView := views.NewView([]string{"amcache_application", "amcache_application_file"},
		`CREATE VIEW V_AmcacheApplications AS
		SELECT 
		    T2.last_write_time,
		    T2.program_id,
		    T2.file_id,
		    T2.lower_case_long_path,
		    T2.name,
		    T2.original_file_name,
		    T2.publisher,
		    T2.version,
		    T2.bin_file_version,
		    T2.binary_type,
		    T2.product_name,
		    T2.product_version,
		    T2.link_date,
		    T2.bin_product_version,
		    T2.size,
		    T2.language,
		    T2.usn,
		    T2.appx_package_full_name,
		    T2.is_os_component,
		    T2.appx_package_relative_id,
		    T1.program_instance_id,
		    T1.install_date,
		    T1.source,
		    T1.root_dir_path,
		    T1.hidden_arp,
		    T1.uninstall_string,
		    T1.registry_key_path,
		    T1.store_app_type,
		    T1.inbox_modern_app,
		    T1.manifest_path,
		    T1.package_full_name,
		    T1.msi_package_code,
		    T1.msi_product_code,
		    T1.msi_install_date,
		    T1.bundle_manifest_path,
		    T1.user_sid
		FROM amcache_application_file T2
		LEFT JOIN amcache_application T1 ON T2.program_id = T1.program_id;`)

	viewsToCreate := []*views.View{applicationsView}
	views.CreateViews(socket, viewsToCreate)
}
