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

type DevicePnpEntry struct {
	LastWriteTime           int64  `json:"last_write_time,string"`
	Model                   string `json:"model"`
	Manufacturer            string `json:"manufacturer"`
	DriverName              string `json:"driver_name"`
	ParentId                string `json:"parent_id"`
	MatchingID              string `json:"matching_id"`
	Class                   string `json:"class"`
	ClassGuid               string `json:"class_guid"`
	Description             string `json:"description"`
	Enumerator              string `json:"enumerator"`
	Service                 string `json:"service"`
	InstallState            string `json:"install_state"`
	DeviceState             string `json:"device_state"`
	Inf                     string `json:"inf"`
	DriverVerDate           string `json:"driver_ver_date"`
	InstallDate             string `json:"install_date"`
	FirstInstallDate        string `json:"first_install_date"`
	DriverPackageStrongName string `json:"driver_package_strong_name"`
	DriverVerVersion        string `json:"driver_ver_version"`
	ContainerId             string `json:"container_id"`
	ProblemCode             string `json:"problem_code"`
	Provider                string `json:"provider"`
	DriverId                string `json:"driver_id"`
	BusReportedDescription  string `json:"bus_reported_description"`
	HWID                    string `json:"hw_id"`
	ExtendedInfs            string `json:"extended_infs"`
	COMPID                  string `json:"compid"`
	STACKID                 string `json:"stack_id"`
	UpperClassFilters       string `json:"upper_class_filters"`
	LowerClassFilters       string `json:"lower_class_filters"`
	UpperFilters            string `json:"upper_filters"`
	LowerFilters            string `json:"lower_filters"`
	DeviceInterfaceClasses  string `json:"device_interface_classes"`
	LocationPaths           string `json:"location_paths"`
}

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

func (dpe *DevicePnpEntry) FieldMappings() map[string]*string {
	return map[string]*string{
		"Model":                   &dpe.Model,
		"Manufacturer":            &dpe.Manufacturer,
		"DriverName":              &dpe.DriverName,
		"ParentId":                &dpe.ParentId,
		"MatchingID":              &dpe.MatchingID,
		"Class":                   &dpe.Class,
		"ClassGuid":               &dpe.ClassGuid,
		"Description":             &dpe.Description,
		"Enumerator":              &dpe.Enumerator,
		"Service":                 &dpe.Service,
		"InstallState":            &dpe.InstallState,
		"DeviceState":             &dpe.DeviceState,
		"Inf":                     &dpe.Inf,
		"DriverVerDate":           &dpe.DriverVerDate,
		"InstallDate":             &dpe.InstallDate,
		"FirstInstallDate":        &dpe.FirstInstallDate,
		"DriverPackageStrongName": &dpe.DriverPackageStrongName,
		"DriverVerVersion":        &dpe.DriverVerVersion,
		"ContainerId":             &dpe.ContainerId,
		"ProblemCode":             &dpe.ProblemCode,
		"Provider":                &dpe.Provider,
		"DriverId":                &dpe.DriverId,
		"BusReportedDescription":  &dpe.BusReportedDescription,
		"HWID":                    &dpe.HWID,
		"ExtendedInfs":            &dpe.ExtendedInfs,
		"COMPID":                  &dpe.COMPID,
		"STACKID":                 &dpe.STACKID,
		"UpperClassFilters":       &dpe.UpperClassFilters,
		"LowerClassFilters":       &dpe.LowerClassFilters,
		"UpperFilters":            &dpe.UpperFilters,
		"LowerFilters":            &dpe.LowerFilters,
		"DeviceInterfaceClasses":  &dpe.DeviceInterfaceClasses,
		"LocationPaths":           &dpe.LocationPaths,
	}
}

func (dpe *DevicePnpEntry) SetLastWriteTime(t int64) {
	dpe.LastWriteTime = t
}

func GetDevicePnpEntriesFromRegistry(registry *regparser.Registry) (map[string][]Entry, error) {
	if registry == nil {
		return nil, fmt.Errorf("registry is nil")
	}

	keyNode := registry.OpenKey(devicePnpKeyPath)
	if keyNode == nil {
		return nil, fmt.Errorf("error opening key: %s", devicePnpKeyPath)
	}

	deviceEntries := make(map[string][]Entry, len(keyNode.Subkeys()))
	for _, subkey := range keyNode.Subkeys() {
		dpe := &DevicePnpEntry{}
		FillInEntryFromKey(dpe, subkey)
		deviceEntries[dpe.DriverId] = append(deviceEntries[dpe.DriverId], dpe)
	}
	return deviceEntries, nil
}

func DevicePnpGenerateFunc(state GlobalStateInterface) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		driverIds := GetConstraintsFromQueryContext("driver_id", queryContext)
		rows := state.GetDevicePnpEntries(driverIds...)
		return RowsAsStringMapArray(rows), nil
	}
}
