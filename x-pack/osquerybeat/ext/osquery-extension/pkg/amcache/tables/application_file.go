// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
	"github.com/osquery/osquery-go/plugin/table"
)

// ApplicationFileEntry represents a single entry in the amcache application file table.
// located at Root\\InventoryApplicationFile
type ApplicationFileEntry struct {
	LastWriteTime         int64  `osquery:"last_write_time"`
	ProgramId             string `osquery:"program_id"`
	FileId                string `osquery:"file_id"`
	LowerCaseLongPath     string `osquery:"lower_case_long_path"`
	Name                  string `osquery:"name"`
	OriginalFileName      string `osquery:"original_file_name"`
	Publisher             string `osquery:"publisher"`
	Version               string `osquery:"version"`
	BinFileVersion        string `osquery:"bin_file_version"`
	BinaryType            string `osquery:"binary_type"`
	ProductName           string `osquery:"product_name"`
	ProductVersion        string `osquery:"product_version"`
	LinkDate              string `osquery:"link_date"`
	BinProductVersion     string `osquery:"bin_product_version"`
	Size                  int64  `osquery:"size"`
	Language              int64  `osquery:"language"`
	Usn                   int64  `osquery:"usn"`
	AppxPackageFullName   string `osquery:"appx_package_full_name"`
	IsOsComponent         string `osquery:"is_os_component"`
	AppxPackageRelativeId string `osquery:"appx_package_relative_id"`
}

// Columns returns the osquery column definitions for the amcache application file table
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
		table.BigIntColumn("size"),
		table.BigIntColumn("language"),
		table.BigIntColumn("usn"),
		table.TextColumn("appx_package_full_name"),
		table.TextColumn("is_os_component"),
		table.TextColumn("appx_package_relative_id"),
	}
}
