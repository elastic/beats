// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
	"strings"
	"time"
)

// ApplicationEntry represents a single entry in the amcache application table.
type ApplicationEntry struct {
	Timestamp          time.Time `osquery:"timestamp" format:"unix"`
	DateTime           time.Time `osquery:"date_time" format:"rfc3339" tz:"UTC"`
	ProgramId          string    `osquery:"program_id"`
	ProgramInstanceId  string    `osquery:"program_instance_id"`
	Name               string    `osquery:"name"`
	Version            string    `osquery:"version"`
	Publisher          string    `osquery:"publisher"`
	Language           int64     `osquery:"language"`
	InstallDate        time.Time `osquery:"install_date"`
	Source             string    `osquery:"source"`
	RootDirPath        string    `osquery:"root_dir_path"`
	HiddenArp          int64     `osquery:"hidden_arp"`
	UninstallString    string    `osquery:"uninstall_string"`
	RegistryKeyPath    string    `osquery:"registry_key_path"`
	StoreAppType       string    `osquery:"store_app_type"`
	InboxModernApp     string    `osquery:"inbox_modern_app"`
	ManifestPath       string    `osquery:"manifest_path"`
	PackageFullName    string    `osquery:"package_full_name"`
	MsiPackageCode     string    `osquery:"msi_package_code"`
	MsiProductCode     string    `osquery:"msi_product_code"`
	MsiInstallDate     time.Time `osquery:"msi_install_date"`
	BundleManifestPath string    `osquery:"bundle_manifest_path"`
	UserSid            string    `osquery:"user_sid"`
	Sha1               string    `osquery:"sha1"`
}

func (e *ApplicationEntry) PostProcess() {
	// The sha1 is the last 40 characters of the ProgramId, the first 4 characters are 0000
	if e.ProgramId == "" || len(e.ProgramId) != 44 {
		return
	}
	e.Sha1 = strings.ToLower(e.ProgramId[4:])
}
