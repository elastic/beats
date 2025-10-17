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

type ApplicationFileEntry struct {
	LastWriteTime         int64  `json:"last_write_time,string"`
	ProgramId             string `json:"program_id"`
	FileId                string `json:"file_id"`
	LowerCaseLongPath     string `json:"lower_case_long_path"`
	Name                  string `json:"name"`
	OriginalFileName      string `json:"original_file_name"`
	Publisher             string `json:"publisher"`
	Version               string `json:"version"`
	BinFileVersion        string `json:"bin_file_version"`
	BinaryType            string `json:"binary_type"`
	ProductName           string `json:"product_name"`
	ProductVersion        string `json:"product_version"`
	LinkDate              string `json:"link_date"`
	BinProductVersion     string `json:"bin_product_version"`
	Size                  string `json:"size"`
	Language              string `json:"language"`
	Usn                   string `json:"usn"`
	AppxPackageFullName   string `json:"appx_package_full_name"`
	IsOsComponent         string `json:"is_os_component"`
	AppxPackageRelativeId string `json:"appx_package_relative_id"`
}

func ApplicationFileColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.BigIntColumn("last_write_time"),
		table.TextColumn("name"),
		table.TextColumn("program_id"),
		table.TextColumn("file_id"),
		table.TextColumn("lower_case_long_path"),
		table.TextColumn("original_file_name"),
		table.TextColumn("publisher"),
		table.TextColumn("version"),
		table.TextColumn("bin_file_version"),
		table.TextColumn("binary_type"),
		table.TextColumn("product_name"),
		table.TextColumn("product_version"),
		table.TextColumn("link_date"),
		table.TextColumn("bin_product_version"),
		table.TextColumn("size"),
		table.TextColumn("language"),
		table.TextColumn("usn"),
		table.TextColumn("appx_package_full_name"),
		table.TextColumn("is_os_component"),
		table.TextColumn("appx_package_relative_id"),
	}
}

func (afe *ApplicationFileEntry) FieldMappings() map[string]*string {
	return map[string]*string{
		"Name":                  &afe.Name,
		"ProgramId":             &afe.ProgramId,
		"FileId":                &afe.FileId,
		"LowerCaseLongPath":     &afe.LowerCaseLongPath,
		"OriginalFileName":      &afe.OriginalFileName,
		"Publisher":             &afe.Publisher,
		"Version":               &afe.Version,
		"BinFileVersion":        &afe.BinFileVersion,
		"BinaryType":            &afe.BinaryType,
		"ProductName":           &afe.ProductName,
		"ProductVersion":        &afe.ProductVersion,
		"LinkDate":              &afe.LinkDate,
		"BinProductVersion":     &afe.BinProductVersion,
		"Size":                  &afe.Size,
		"Language":              &afe.Language,
		"Usn":                   &afe.Usn,
		"AppxPackageFullName":   &afe.AppxPackageFullName,
		"IsOsComponent":         &afe.IsOsComponent,
		"AppxPackageRelativeId": &afe.AppxPackageRelativeId,
	}
}

func (afe *ApplicationFileEntry) SetLastWriteTime(t int64) {
	afe.LastWriteTime = t
}

func GetApplicationFileEntriesFromRegistry(registry *regparser.Registry) (map[string][]Entry, error) {
	if registry == nil {
		return nil, fmt.Errorf("registry is nil")
	}

	keyNode := registry.OpenKey(applicationFileKeyPath)
	if keyNode == nil {
		return nil, fmt.Errorf("error opening key: %s", applicationFileKeyPath)
	}

	applicationEntries := make(map[string][]Entry, len(keyNode.Subkeys()))
	for _, subkey := range keyNode.Subkeys() {
		ae := &ApplicationFileEntry{}
		FillInEntryFromKey(ae, subkey)
		applicationEntries[ae.ProgramId] = append(applicationEntries[ae.ProgramId], ae)
	}
	return applicationEntries, nil
}

func ApplicationFileGenerateFunc(state GlobalStateInterface) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		programIds := GetConstraintsFromQueryContext("program_id", queryContext)
		rows := state.GetApplicationFileEntries(programIds...)
		return RowsAsStringMapArray(rows), nil
	}
}
