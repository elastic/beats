// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/encoding"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/filters"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	"github.com/osquery/osquery-go/plugin/table"
	"www.velocidex.com/golang/regparser"
)

// GlobalState is an interface that defines methods for accessing global Amcache state.
type GlobalStateInterface interface {
	GetCachedEntries(amcacheTable AmcacheTable, filters []filters.Filter, log *logger.Logger) []Entry
}

type Entry interface {
	// PostProcess is called after the entry is populated from the registry key.
	// It is used to perform any additional processing on the entry, such as converting
	// the timestamp to a more human-readable format.
	PostProcess()
}

type AmcacheTable struct {
	Name     string
	HiveKey  string
	NewEntry func() Entry
}

// AllAmcacheTables returns a slice of all defined AmcacheTables.
func AllAmcacheTables() []AmcacheTable {
	return []AmcacheTable{
		{
			Name:    "amcache_application",
			HiveKey: "Root\\InventoryApplication",
			NewEntry: func() Entry {
				return &ApplicationEntry{}
			},
		},
		{
			Name:    "amcache_application_file",
			HiveKey: "Root\\InventoryApplicationFile",
			NewEntry: func() Entry {
				return &ApplicationFileEntry{}
			},
		},
		{
			Name:    "amcache_application_shortcut",
			HiveKey: "Root\\InventoryApplicationShortcut",
			NewEntry: func() Entry {
				return &ApplicationShortcutEntry{}
			},
		},
		{
			Name:    "amcache_driver_binary",
			HiveKey: "Root\\InventoryDriverBinary",
			NewEntry: func() Entry {
				return &DriverBinaryEntry{}
			},
		},
		{
			Name:    "amcache_device_pnp",
			HiveKey: "Root\\InventoryDevicePnp",
			NewEntry: func() Entry {
				return &DevicePnpEntry{}
			},
		},
		{
			Name:    "amcache_driver_package",
			HiveKey: "Root\\InventoryDriverPackage",
			NewEntry: func() Entry {
				return &DriverPackageEntry{}
			},
		},
	}
}

func (t AmcacheTable) Columns() []table.ColumnDefinition {
	entry := t.NewEntry()
	columns, err := encoding.GenerateColumnDefinitions(entry)
	if err != nil {
		panic(fmt.Sprintf("failed to generate column definitions for %s: %v", t, err))
	}
	return columns
}

func (t AmcacheTable) GenerateFunc(state GlobalStateInterface, log *logger.Logger) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		filters := filters.GetConstraintFilters(queryContext)
		entries := state.GetCachedEntries(t, filters, log)
		marshalled, err := MarshalEntries(entries)
		if err != nil {
			return nil, err
		}
		return marshalled, nil
	}
}

// MarshalEntries takes a slice of Entry interfaces and marshals each to a map[string]string.
func MarshalEntries(entries []Entry) ([]map[string]string, error) {
	marshalled := make([]map[string]string, 0, len(entries))
	for _, entry := range entries {
		mapped, err := encoding.MarshalToMap(entry)
		if err != nil {
			return nil, err
		}
		marshalled = append(marshalled, mapped)
	}
	return marshalled, nil
}

func SetEntryCommonFields(e Entry, key *regparser.CM_KEY_NODE) {
	elem := reflect.ValueOf(e).Elem()
	if !elem.IsValid() || !elem.CanSet() {
		return
	}

	// Set KeyName from key name
	keyName := elem.FieldByName("KeyName")
	if keyName.IsValid() && keyName.CanSet() {
		keyName.Set(reflect.ValueOf(key.Name()))
	}

	// Set Timestamp from key timestamp
	timestamp := elem.FieldByName("Timestamp")
	if timestamp.IsValid() && timestamp.CanSet() {
		timestamp.Set(reflect.ValueOf(key.LastWriteTime().Local()))
	}
}

// FillInEntryFromKey takes an any, and using the FieldMappings, populates its fields from a registry key.
func FillInEntryFromKey(e Entry, key *regparser.CM_KEY_NODE) {
	elem := reflect.ValueOf(e).Elem()
	if !elem.IsValid() || !elem.CanSet() {
		return
	}

	for _, value := range key.Values() {
		if value.ValueName() == "" {
			continue
		}
		field := elem.FieldByName(value.ValueName())
		if !field.IsValid() || !field.CanSet() {
			continue
		}

		switch field.Kind() {

		// If the field is an integer type, make sure the registry value is DWORD or QWORD
		case reflect.Int64, reflect.Int32:
			switch value.ValueData().Type {
			case regparser.REG_DWORD, regparser.REG_QWORD:
				field.SetInt(int64(value.ValueData().Uint64))
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
			//log.Printf("Warning: unsupported field type for %s: %s", value.ValueName(), field.Kind())
		}
	}
	// Set the common fields for the entry, timestamp and key name
	SetEntryCommonFields(e, key)

	// Call the PostProcess method to perform any additional processing on the entry
	e.PostProcess()
}

// GetEntriesFromRegistry reads the registry and returns a map of entries for the specified TableType.
func GetEntriesFromRegistry(amcacheTable AmcacheTable, registry *regparser.Registry) ([]Entry, error) {
	if registry == nil {
		return nil, fmt.Errorf("registry is nil")
	}

	hiveKey := amcacheTable.HiveKey
	keyNode := registry.OpenKey(hiveKey)
	if keyNode == nil {
		return nil, fmt.Errorf("error opening key: %s", hiveKey)
	}

	entries := make([]Entry, 0, len(keyNode.Subkeys()))
	for _, subkey := range keyNode.Subkeys() {
		ae := amcacheTable.NewEntry()
		FillInEntryFromKey(ae, subkey)
		entries = append(entries, ae)
	}
	return entries, nil
}
