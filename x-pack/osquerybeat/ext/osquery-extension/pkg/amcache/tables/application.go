// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import "time"

// ApplicationEntry represents a single entry in the amcache application table.
type ApplicationEntry struct {
	Timestamp          time.Time `osquery:"timestamp" format:"unix"`
	DateTime           time.Time `osquery:"date_time" format:"rfc3339" tz:"UTC"`
	KeyName            string    `osquery:"key_name"`
	ProgramId          string    `osquery:"program_id"`
	ProgramInstanceId  string    `osquery:"program_instance_id"`
	Name               string    `osquery:"name"`
	Version            string    `osquery:"version"`
	Publisher          string    `osquery:"publisher"`
	Language           int64     `osquery:"language"`
	InstallDate        string    `osquery:"install_date"`
	Source             string    `osquery:"source"`
	RootDirPath        string    `osquery:"root_dir_path"`
	HiddenArp          string    `osquery:"hidden_arp"`
	UninstallString    string    `osquery:"uninstall_string"`
	RegistryKeyPath    string    `osquery:"registry_key_path"`
	StoreAppType       string    `osquery:"store_app_type"`
	InboxModernApp     string    `osquery:"inbox_modern_app"`
	ManifestPath       string    `osquery:"manifest_path"`
	PackageFullName    string    `osquery:"package_full_name"`
	MsiPackageCode     string    `osquery:"msi_package_code"`
	MsiProductCode     string    `osquery:"msi_product_code"`
	MsiInstallDate     string    `osquery:"msi_install_date"`
	BundleManifestPath string    `osquery:"bundle_manifest_path"`
	UserSid            string    `osquery:"user_sid"`
}
