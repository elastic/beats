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

// DriverBinaryEntry represents a single entry in the amcache driver binary table.
type DriverBinaryEntry struct {
	LastWriteTime           int64  `osquery:"last_write_time"`
	DriverName              string `osquery:"driver_name"`
	Inf                     string `osquery:"inf"`
	DriverVersion           string `osquery:"driver_version"`
	Product                 string `osquery:"product"`
	ProductVersion          string `osquery:"product_version"`
	WdfVersion              string `osquery:"wdf_version"`
	DriverCompany           string `osquery:"driver_company"`
	DriverPackageStrongName string `osquery:"driver_package_strong_name"`
	Service                 string `osquery:"service"`
	DriverInBox             string `osquery:"driver_in_box"`
	DriverSigned            string `osquery:"driver_signed"`
	DriverIsKernelMode      string `osquery:"driver_is_kernel_mode"`
	DriverId                string `osquery:"driver_id"`
	DriverLastWriteTime     string `osquery:"driver_last_write_time"`
	DriverType              string `osquery:"driver_type"`
	DriverTimeStamp         string `osquery:"driver_time_stamp"`
	DriverCheckSum          string `osquery:"driver_check_sum"`
	ImageSize               string `osquery:"image_size"`
}

// FilterColumn returns the name of the column used for filtering entries in the DriverBinaryTable.
func (dbe *DriverBinaryEntry) FilterValue() string {
	return dbe.DriverId
}

// ToMap converts the DriverBinaryEntry to a map[string]string representation.
func (dbe *DriverBinaryEntry) ToMap() (map[string]string, error) {
	mapped, err := encoding.MarshalToMap(dbe)
	return mapped, err
}

// DriverBinaryTable implements the TableInterface for the amcache driver binary table.
type DriverBinaryTable struct {}

// Type returns the TableType for the DriverBinaryTable.
func (dbt *DriverBinaryTable) Type() TableType {
	return DriverBinaryTableType
}

// FilterColumn returns the name of the column used for filtering entries in the DriverBinaryTable.
func (dbt *DriverBinaryTable) FilterColumn() string {
	return "driver_id"
}

// Columns returns the column definitions for the DriverBinaryTable.
func (dbt *DriverBinaryTable) Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.BigIntColumn("last_write_time"),
		table.TextColumn("driver_name"),
		table.TextColumn("inf"),
		table.TextColumn("driver_version"),
		table.TextColumn("product"),
		table.TextColumn("product_version"),
		table.TextColumn("wdf_version"),
		table.TextColumn("driver_company"),
		table.TextColumn("driver_package_strong_name"),
		table.TextColumn("service"),
		table.TextColumn("driver_in_box"),
		table.TextColumn("driver_signed"),
		table.TextColumn("driver_is_kernel_mode"),
		table.TextColumn("driver_id"),
	}
}

// GetID returns the unique identifier for the DriverBinaryEntry, which is the DriverId.
func (ae *DriverBinaryEntry) GetID() string {
	return ae.DriverId
}

// GetType returns the TableType for the DriverBinaryEntry.
func (dbt *DriverBinaryTable) GenerateFunc(state GlobalStateInterface) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		driverIds := GetConstraintsFromQueryContext(dbt.FilterColumn(), queryContext)
		entries := state.GetCachedEntries(dbt.Type(), driverIds...)

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
