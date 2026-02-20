// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"www.velocidex.com/golang/regparser"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

type Entry interface {
	// PostProcess is called after the entry is populated from the registry key.
	// It is used to perform any additional processing on the entry such as
	// calculating derived fields or performing other cleanup.
	PostProcess()
}

type TableName string

const (
	TableNameApplication         TableName = "elastic_amcache_application"
	TableNameApplicationFile     TableName = "elastic_amcache_application_file"
	TableNameApplicationShortcut TableName = "elastic_amcache_application_shortcut"
	TableNameDriverBinary        TableName = "elastic_amcache_driver_binary"
	TableNameDevicePnp           TableName = "elastic_amcache_device_pnp"
	TableNameDriverPackage       TableName = "elastic_amcache_driver_package"
)

type AmcacheTable struct {
	Name     TableName
	hiveKey  string
	newEntry func() Entry
}

// AllAmcacheTables returns a slice of all defined AmcacheTables.
func AllAmcacheTables() []AmcacheTable {
	return []AmcacheTable{
		{
			Name:    TableNameApplication,
			hiveKey: "Root\\InventoryApplication",
			newEntry: func() Entry {
				return &ApplicationEntry{}
			},
		},
		{
			Name:    TableNameApplicationFile,
			hiveKey: "Root\\InventoryApplicationFile",
			newEntry: func() Entry {
				return &ApplicationFileEntry{}
			},
		},
		{
			Name:    TableNameApplicationShortcut,
			hiveKey: "Root\\InventoryApplicationShortcut",
			newEntry: func() Entry {
				return &ApplicationShortcutEntry{}
			},
		},
		{
			Name:    TableNameDriverBinary,
			hiveKey: "Root\\InventoryDriverBinary",
			newEntry: func() Entry {
				return &DriverBinaryEntry{}
			},
		},
		{
			Name:    TableNameDevicePnp,
			hiveKey: "Root\\InventoryDevicePnp",
			newEntry: func() Entry {
				return &DevicePnpEntry{}
			},
		},
		{
			Name:    TableNameDriverPackage,
			hiveKey: "Root\\InventoryDriverPackage",
			newEntry: func() Entry {
				return &DriverPackageEntry{}
			},
		},
	}
}

func GetAmcacheTableByName(name TableName) *AmcacheTable {
	for _, table := range AllAmcacheTables() {
		if table.Name == name {
			return &table
		}
	}
	return nil
}

// fillInEntryFromKey populates the fields of an Entry from a registry key.
func fillInEntryFromKey(e Entry, key *regparser.CM_KEY_NODE, log *logger.Logger) {
	// Get the element of the entry and make sure it is valid and can be set
	elem := reflect.ValueOf(e).Elem()
	if !elem.IsValid() || !elem.CanSet() {
		return
	}

	// iterate over the values in the key
	for _, value := range key.Values() {
		// Skip if the value name is empty
		if value.ValueName() == "" {
			continue
		}

		// Get the field by the value name
		// this function hinges on the entry having a field
		// with the same name as the value name
		field := elem.FieldByName(value.ValueName())
		if !field.IsValid() || !field.CanSet() {
			continue
		}

		// Switch on the kind of the field
		switch field.Kind() {

		// If the field is an integer type, make sure the registry value is DWORD or QWORD
		case reflect.Int64:
			switch value.ValueData().Type {
			case regparser.REG_DWORD, regparser.REG_QWORD:
				val := value.ValueData().Uint64
				safeUint64ToInt64 := func(val uint64) int64 {
					if val > math.MaxInt64 {
						return math.MaxInt64
					}
					return int64(val)
				}
				// all integers in osquery are 64bit signed integers, but all registry values are unsigned
				// so we need to convert the value to an int64 and check for overflow
				field.SetInt(safeUint64ToInt64(val))
			}
		// If the field is a string type, handle STRING, DWORD, and QWORD registry value types
		case reflect.String:
			switch value.ValueData().Type {
			case regparser.REG_SZ:
				field.SetString(strings.TrimRight(value.ValueData().String, "\x00"))
			case regparser.REG_DWORD, regparser.REG_QWORD:
				field.SetString(fmt.Sprintf("%d", value.ValueData().Uint64))
			}

		// If the field is a boolean type, interpret non-zero DWORD/QWORD as true
		case reflect.Bool:
			if value.ValueData().Uint64 != 0 {
				field.SetBool(true)
			} else {
				field.SetBool(false)
			}
		case reflect.Struct:
			switch field.Type() {
			case reflect.TypeOf(time.Time{}):
				timeString := strings.TrimRight(value.ValueData().String, "\x00")
				timestamp, err := time.Parse("01/02/2006 15:04:05", timeString)
				if err != nil {
					continue
				}
				field.Set(reflect.ValueOf(timestamp))
			}
		// Unsupported field type
		default:
			// We control the entry types, so this should never happen
			panic(fmt.Sprintf("Error: unsupported field type for %s: %s", value.ValueName(), field.Kind()))
		}
	}

	// Set the timestamp for the entry
	elem.FieldByName("Timestamp").Set(reflect.ValueOf(key.LastWriteTime().Time))
	elem.FieldByName("DateTime").Set(reflect.ValueOf(key.LastWriteTime().Local()))

	// Call the PostProcess method to perform any additional processing on the entry
	e.PostProcess()
}

// GetEntriesFromRegistry reads the registry and returns a slice of entries for the specified AmcacheTable.
func GetEntriesFromRegistry(amcacheTable AmcacheTable, registry *regparser.Registry, log *logger.Logger) ([]Entry, error) {
	if registry == nil {
		return nil, fmt.Errorf("GetEntriesFromRegistry called with nil registry for table %s", amcacheTable.Name)
	}

	hiveKey := amcacheTable.hiveKey
	keyNode := registry.OpenKey(hiveKey)
	if keyNode == nil {
		return nil, fmt.Errorf("failed to open key: %s", hiveKey)
	}

	entries := make([]Entry, 0, len(keyNode.Subkeys()))
	for _, subkey := range keyNode.Subkeys() {
		ae := amcacheTable.newEntry()
		fillInEntryFromKey(ae, subkey, log)
		entries = append(entries, ae)
	}
	return entries, nil
}
