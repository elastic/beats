// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
	"github.com/osquery/osquery-go/plugin/table"
)

// ApplicationEntry represents a single entry in the amcache application table.
type ApplicationEntry struct {
	LastWriteTime      int64  `osquery:"last_write_time"`
	ProgramId          string `osquery:"program_id"`
	ProgramInstanceId  string `osquery:"program_instance_id"`
	Name               string `osquery:"name"`
	Version            string `osquery:"version"`
	Publisher          string `osquery:"publisher"`
	Language           int64  `osquery:"language"`
	InstallDate        string `osquery:"install_date"`
	Source             string `osquery:"source"`
	RootDirPath        string `osquery:"root_dir_path"`
	HiddenArp          string `osquery:"hidden_arp"`
	UninstallString    string `osquery:"uninstall_string"`
	RegistryKeyPath    string `osquery:"registry_key_path"`
	StoreAppType       string `osquery:"store_app_type"`
	InboxModernApp     string `osquery:"inbox_modern_app"`
	ManifestPath       string `osquery:"manifest_path"`
	PackageFullName    string `osquery:"package_full_name"`
	MsiPackageCode     string `osquery:"msi_package_code"`
	MsiProductCode     string `osquery:"msi_product_code"`
	MsiInstallDate     string `osquery:"msi_install_date"`
	BundleManifestPath string `osquery:"bundle_manifest_path"`
	UserSid            string `osquery:"user_sid"`
}

// Columns returns the column definitions for the ApplicationTable.
func ApplicationColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.BigIntColumn("last_write_time"),
		table.TextColumn("name"),
		table.TextColumn("program_id"),
		table.TextColumn("program_instance_id"),
		table.TextColumn("version"),
		table.TextColumn("publisher"),
		table.TextColumn("language"),
		table.TextColumn("install_date"),
		table.TextColumn("source"),
		table.TextColumn("root_dir_path"),
		table.TextColumn("hidden_arp"),
		table.TextColumn("uninstall_string"),
		table.TextColumn("registry_key_path"),
		table.TextColumn("store_app_type"),
		table.TextColumn("inbox_modern_app"),
		table.TextColumn("manifest_path"),
		table.TextColumn("package_full_name"),
		table.TextColumn("msi_package_code"),
		table.TextColumn("msi_product_code"),
		table.TextColumn("msi_install_date"),
		table.TextColumn("bundle_manifest_path"),
		table.TextColumn("user_sid"),
	}
}
