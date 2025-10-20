// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
	"context"
	"fmt"
	"github.com/osquery/osquery-go/plugin/table"
	"www.velocidex.com/golang/regparser"
)

type ApplicationEntry struct {
	LastWriteTime      int64  `json:"last_write_time,string"`
	ProgramId          string `json:"program_id"`
	ProgramInstanceId  string `json:"program_instance_id"`
	Name               string `json:"name"`
	Version            string `json:"version"`
	Publisher          string `json:"publisher"`
	Language           string `json:"language"`
	InstallDate        string `json:"install_date"`
	Source             string `json:"source"`
	RootDirPath        string `json:"root_dir_path"`
	HiddenArp          string `json:"hidden_arp"`
	UninstallString    string `json:"uninstall_string"`
	RegistryKeyPath    string `json:"registry_key_path"`
	StoreAppType       string `json:"store_app_type"`
	InboxModernApp     string `json:"inbox_modern_app"`
	ManifestPath       string `json:"manifest_path"`
	PackageFullName    string `json:"package_full_name"`
	MsiPackageCode     string `json:"msi_package_code"`
	MsiProductCode     string `json:"msi_product_code"`
	MsiInstallDate     string `json:"msi_install_date"`
	BundleManifestPath string `json:"bundle_manifest_path"`
	UserSid            string `json:"user_sid"`
}

func (ae *ApplicationEntry) FieldMappings() map[string]*string {
	return map[string]*string{
		"Name":               &ae.Name,
		"ProgramId":          &ae.ProgramId,
		"ProgramInstanceId":  &ae.ProgramInstanceId,
		"Version":            &ae.Version,
		"Publisher":          &ae.Publisher,
		"Language":           &ae.Language,
		"InstallDate":        &ae.InstallDate,
		"Source":             &ae.Source,
		"RootDirPath":        &ae.RootDirPath,
		"HiddenArp":          &ae.HiddenArp,
		"UninstallString":    &ae.UninstallString,
		"RegistryKeyPath":    &ae.RegistryKeyPath,
		"StoreAppType":       &ae.StoreAppType,
		"InboxModernApp":     &ae.InboxModernApp,
		"ManifestPath":       &ae.ManifestPath,
		"PackageFullName":    &ae.PackageFullName,
		"MsiPackageCode":     &ae.MsiPackageCode,
		"MsiProductCode":     &ae.MsiProductCode,
		"MsiInstallDate":     &ae.MsiInstallDate,
		"BundleManifestPath": &ae.BundleManifestPath,
		"UserSid":            &ae.UserSid,
	}
}

func (ae *ApplicationEntry) SetLastWriteTime(t int64) {
	ae.LastWriteTime = t
}

func GetApplicationEntriesFromRegistry(registry *regparser.Registry) (map[string][]Entry, error) {
	if registry == nil {
		return nil, fmt.Errorf("registry is nil")
	}

	keyNode := registry.OpenKey(applicationKeyPath)
	if keyNode == nil {
		return nil, fmt.Errorf("error opening key: %s", applicationKeyPath)
	}

	applicationEntries := make(map[string][]Entry, len(keyNode.Subkeys()))
	for _, subkey := range keyNode.Subkeys() {
		ae := &ApplicationEntry{}
		FillInEntryFromKey(ae, subkey)

		applicationEntries[ae.ProgramId] = append(applicationEntries[ae.ProgramId], ae)
	}
	return applicationEntries, nil
}

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

func ApplicationGenerateFunc(state GlobalStateInterface) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		programIds := GetConstraintsFromQueryContext("program_id", queryContext)
		rows := state.GetApplicationEntries(programIds...)
		return RowsAsStringMapArray(rows), nil
	}
}
