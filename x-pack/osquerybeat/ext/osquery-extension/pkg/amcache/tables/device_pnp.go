// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
	"github.com/osquery/osquery-go/plugin/table"
)

// DevicePnpEntry represents a single entry in the amcache device pnp table.
// located at Root\InventoryDevicePnp
type DevicePnpEntry struct {
	LastWriteTime           int64  `osquery:"last_write_time"`
	Model                   string `osquery:"model"`
	Manufacturer            string `osquery:"manufacturer"`
	DriverName              string `osquery:"driver_name"`
	ParentId                string `osquery:"parent_id"`
	MatchingID              string `osquery:"matching_id"`
	Class                   string `osquery:"class"`
	ClassGuid               string `osquery:"class_guid"`
	Description             string `osquery:"description"`
	Enumerator              string `osquery:"enumerator"`
	Service                 string `osquery:"service"`
	InstallState            string `osquery:"install_state"`
	DeviceState             string `osquery:"device_state"`
	Inf                     string `osquery:"inf"`
	DriverVerDate           string `osquery:"driver_ver_date"`
	InstallDate             string `osquery:"install_date"`
	FirstInstallDate        string `osquery:"first_install_date"`
	DriverPackageStrongName string `osquery:"driver_package_strong_name"`
	DriverVerVersion        string `osquery:"driver_ver_version"`
	ContainerId             string `osquery:"container_id"`
	ProblemCode             string `osquery:"problem_code"`
	Provider                string `osquery:"provider"`
	DriverId                string `osquery:"driver_id"`
	BusReportedDescription  string `osquery:"bus_reported_description"`
	HWID                    string `osquery:"hw_id"`
	ExtendedInfs            string `osquery:"extended_infs"`
	COMPID                  string `osquery:"compid"`
	STACKID                 string `osquery:"stack_id"`
	UpperClassFilters       string `osquery:"upper_class_filters"`
	LowerClassFilters       string `osquery:"lower_class_filters"`
	UpperFilters            string `osquery:"upper_filters"`
	LowerFilters            string `osquery:"lower_filters"`
	DeviceInterfaceClasses  string `osquery:"device_interface_classes"`
	LocationPaths           string `osquery:"location_paths"`
}

// Columns returns the column definitions for the DevicePnpTable.
func DevicePnpColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.BigIntColumn("last_write_time"),
		table.TextColumn("model"),
		table.TextColumn("manufacturer"),
		table.TextColumn("driver_name"),
		table.TextColumn("parent_id"),
		table.TextColumn("matching_id"),
		table.TextColumn("class"),
		table.TextColumn("class_guid"),
		table.TextColumn("description"),
		table.TextColumn("enumerator"),
		table.TextColumn("service"),
		table.TextColumn("install_state"),
		table.TextColumn("device_state"),
		table.TextColumn("inf"),
		table.TextColumn("driver_ver_date"),
		table.TextColumn("install_date"),
		table.TextColumn("first_install_date"),
		table.TextColumn("driver_package_strong_name"),
		table.TextColumn("driver_ver_version"),
		table.TextColumn("container_id"),
		table.TextColumn("problem_code"),
		table.TextColumn("provider"),
		table.TextColumn("driver_id"),
		table.TextColumn("bus_reported_description"),
		table.TextColumn("hw_id"),
		table.TextColumn("extended_infs"),
		table.TextColumn("compid"),
		table.TextColumn("stack_id"),
		table.TextColumn("upper_class_filters"),
		table.TextColumn("lower_class_filters"),
		table.TextColumn("upper_filters"),
		table.TextColumn("lower_filters"),
		table.TextColumn("device_interface_classes"),
		table.TextColumn("location_paths"),
	}
}
