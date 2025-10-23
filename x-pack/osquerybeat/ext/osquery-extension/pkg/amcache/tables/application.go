// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/encoding"
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

// FilterValue returns the index value for the ApplicationEntry, which is the ProgramId.
func (ae *ApplicationEntry) FilterValue() string {
	return ae.ProgramId
}

// ToMap converts the ApplicationEntry to a map[string]string representation.
func (ae *ApplicationEntry) ToMap() (map[string]string, error) {
	mapped, err := encoding.MarshalToMap(ae)
	return mapped, err
}

// ApplicationTable implements the TableInterface for the amcache application table.
type ApplicationTable struct{}

// Type returns the TableType for the ApplicationTable.
func (at *ApplicationTable) Type() TableType {
	return ApplicationTableType
}

// FilterColumn returns the name of the column used for filtering entries in the ApplicationTable.
func (at *ApplicationTable) FilterColumn() string {
	return "program_id"
}

// Columns returns the column definitions for the ApplicationTable.
func (at *ApplicationTable) Columns() []table.ColumnDefinition {
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

// GenerateFunc generates the data for the ApplicationTable based on the provided GlobalStateInterface.
func (at *ApplicationTable) GenerateFunc(state GlobalStateInterface) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		programIds := GetConstraintsFromQueryContext(at.FilterColumn(), queryContext)
		entries := state.GetCachedEntries(at.Type(), programIds...)
		rows := make([]map[string]string, 0, len(entries))
		for _, entry := range entries {
			mapped, err := entry.ToMap()
			if err != nil {
				return nil, err
			}
			rows = append(rows, mapped)
		}
		return rows, nil
	}
}
