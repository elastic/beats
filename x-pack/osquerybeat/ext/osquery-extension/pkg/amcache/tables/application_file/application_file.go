// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package application_file

import (
	"context"
	"fmt"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/interfaces"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/utilities"
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

type ApplicationFileTable struct {
	Entries []interfaces.Entry
}

func (t *ApplicationFileTable) AddRow(key *regparser.CM_KEY_NODE) error {
	afe := &ApplicationFileEntry{}
	interfaces.FillInEntryFromKey(afe, key)
	t.Entries = append(t.Entries, afe)
	return nil
}

func (t *ApplicationFileTable) Rows() []interfaces.Entry {
	return t.Entries
}

func (t *ApplicationFileTable) KeyName() string {
	return "Root\\InventoryApplicationFile"
}

func GenerateFunc(hiveReader *utilities.HiveReader) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		table := ApplicationFileTable{}
		err := interfaces.BuildTableFromRegistry(&table, hiveReader, ctx, queryContext)
		if err != nil {
			return nil, fmt.Errorf("failed to build ApplicationFileTable: %w", err)
		}
		return interfaces.RowsAsStringMapArray(&table), nil
	}
}
