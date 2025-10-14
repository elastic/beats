// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package driver_binary

import (
	"context"
	"fmt"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/interfaces"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/utilities"
	"github.com/osquery/osquery-go/plugin/table"
	"www.velocidex.com/golang/regparser"
)

type DriverBinaryEntry struct {
	LastWriteTime int64  `json:"last_write_time,string"`
    DriverName string `json:"driver_name"`
    Inf string `json:"inf"`
    DriverVersion string `json:"driver_version"`
    Product string `json:"product"`
    ProductVersion string `json:"product_version"`
    WdfVersion string `json:"wdf_version"`
    DriverCompany string `json:"driver_company"`
    DriverPackageStrongName string `json:"driver_package_strong_name"`
    Service string `json:"service"`
    DriverInBox string `json:"driver_in_box"`
    DriverSigned string `json:"driver_signed"`
    DriverIsKernelMode string `json:"driver_is_kernel_mode"`
    DriverId string `json:"driver_id"`
    DriverLastWriteTime string `json:"driver_last_write_time"`
    DriverType string `json:"driver_type"`
    DriverTimeStamp string `json:"driver_time_stamp"`
    DriverCheckSum string `json:"driver_check_sum"`
    ImageSize string `json:"image_size"`
}

func DriverBinaryColumns() []table.ColumnDefinition {
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

func (ae *DriverBinaryEntry) FieldMappings() map[string]*string {
	return map[string]*string{
		"DriverName":              &ae.DriverName,
		"Inf":                     &ae.Inf,
		"DriverVersion":           &ae.DriverVersion,
		"Product":                 &ae.Product,
		"ProductVersion":          &ae.ProductVersion,
		"WdfVersion":              &ae.WdfVersion,
		"DriverCompany":           &ae.DriverCompany,
		"DriverPackageStrongName": &ae.DriverPackageStrongName,
		"Service":                 &ae.Service,
		"DriverInBox":             &ae.DriverInBox,
		"DriverSigned":            &ae.DriverSigned,
		"DriverIsKernelMode":      &ae.DriverIsKernelMode,
		"DriverId":                &ae.DriverId,
	}
}

func (ae *DriverBinaryEntry) SetLastWriteTime(t int64) {
	ae.LastWriteTime = t
}

type DriverBinaryTable struct {
	Entries []interfaces.Entry
}

func (t *DriverBinaryTable) AddRow(key *regparser.CM_KEY_NODE) error {
	ae := &DriverBinaryEntry{}
	interfaces.FillInEntryFromKey(ae, key)
	t.Entries = append(t.Entries, ae)
	return nil
}

func (t *DriverBinaryTable) Rows() []interfaces.Entry {
	return t.Entries
}

func (t *DriverBinaryTable) KeyName() string {
	return "Root\\InventoryDriverBinary"
}

func GenerateFunc(hiveReader *utilities.HiveReader) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		table := DriverBinaryTable{}
		err := interfaces.BuildTableFromRegistry(&table, hiveReader, ctx, queryContext)
		if err != nil {
			return nil, fmt.Errorf("failed to build DriverBinaryTable: %w", err)
		}
		return interfaces.RowsAsStringMapArray(&table), nil
	}
}
