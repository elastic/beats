// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package amcache

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/hooks"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

func AmcacheApplicationView() *hooks.View {
	return hooks.NewView(
		"elastic_amcache_applications_view",
		[]string{"elastic_amcache_application", "elastic_amcache_application_file"},
		`CREATE VIEW elastic_amcache_applications_view AS
		-- Part 1: Get all 'app' rows, and any matching 'file' rows
		SELECT
			-- 'file' columns (22) - Prioritize file data when it exists
			COALESCE(file.timestamp, app.timestamp) AS timestamp,
			COALESCE(file.date_time, app.date_time) AS date_time,
			app.program_id, -- Use app.program_id as the anchor
			file.file_id,
			file.lower_case_long_path,
			COALESCE(file.name, app.name) AS name,
			file.original_file_name,
			COALESCE(file.publisher, app.publisher) AS publisher,
			COALESCE(file.version, app.version) AS version,
			file.bin_file_version,
			file.binary_type,
			file.product_name,
			file.product_version,
			file.link_date,
			file.bin_product_version,
			file.size,
			COALESCE(file.language, app.language) AS language,
			file.usn,
			file.appx_package_full_name,
			file.is_os_component,
			file.appx_package_relative_id,
			file.sha1 AS file_sha1,
		
			-- 'app'-Only columns (16)
			app.program_instance_id,
			app.install_date,
			app.source,
			app.root_dir_path,
			app.hidden_arp,
			app.uninstall_string,
			app.registry_key_path,
			app.store_app_type,
			app.inbox_modern_app,
			app.manifest_path,
			app.package_full_name,
			app.msi_package_code,
			app.msi_product_code,
			app.msi_install_date,
			app.bundle_manifest_path,
			app.user_sid,
			app.sha1 as app_sha1
		FROM
			elastic_amcache_application AS app
		LEFT JOIN elastic_amcache_application_file AS file ON app.program_id = file.program_id

		UNION ALL

		-- Part 2: Get all 'file' rows that had NO 'app' match
		SELECT
			-- 'file' columns (22) - These all have data
			file.timestamp,
			file.date_time,
			file.program_id,
			file.file_id,
			file.lower_case_long_path,
			file.name,
			file.original_file_name,
			file.publisher,
			file.version,
			file.bin_file_version,
			file.binary_type,
			file.product_name,
			file.product_version,
			file.link_date,
			file.bin_product_version,
			file.size,
			file.language,
			file.usn,
			file.appx_package_full_name,
			file.is_os_component,
			file.appx_package_relative_id,
			file.sha1 AS file_sha1,

			-- 'app'-Only columns (16) - These must all be NULL
			NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL,
			NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL,
			NULL as app_sha1
		FROM
			elastic_amcache_application_file AS file
		LEFT JOIN elastic_amcache_application AS app ON file.program_id = app.program_id
		WHERE
			app.program_id IS NULL;`)
}

func CreateViewHook(socket *string, log *logger.Logger, hookData any) error {
	view, ok := hookData.(*hooks.View)
	if !ok {
		return fmt.Errorf("hook data is not a view")
	}
	return view.Create(socket, log)
}

func DeleteViewHook(socket *string, log *logger.Logger, hookData any) error {
	view, ok := hookData.(*hooks.View)
	if !ok {
		return fmt.Errorf("hook data is not a view")
	}
	return view.Delete(socket, log)
}

func RegisterHooks(hm *hooks.HookManager) {
	amcacheApplicationView := AmcacheApplicationView()
	hm.Register(hooks.NewHook("CreateAmcacheApplicationsView", CreateViewHook, DeleteViewHook, amcacheApplicationView))
}
