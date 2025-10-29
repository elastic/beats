// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
	"context"
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/forensicanalysis/fslib"
	"github.com/forensicanalysis/fslib/systemfs"
	"github.com/osquery/osquery-go/plugin/table"
	"www.velocidex.com/golang/regparser"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/filters"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/encoding"
)

// GlobalState is an interface that defines methods for accessing global Amcache state.
type GlobalStateInterface interface {
	GetCachedEntries(amcacheTable AmcacheTable, filters []filters.Filter) []Entry
}

// Entry defines the methods that each Amcache entry must implement.
type Entry interface {}

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

// GetEntriesFromRegistry reads the registry and returns a map of entries for the specified TableType.
func GetEntriesFromRegistry(amcacheTable AmcacheTable, registry *regparser.Registry) ([]Entry, error) {
	if registry == nil {
		return nil, fmt.Errorf("registry is nil")
	}

	hiveKey := amcacheTable.GetHiveKey()
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

// FillInEntryFromKey takes an any, and using the FieldMappings, populates its fields from a registry key.
func FillInEntryFromKey(e Entry, key *regparser.CM_KEY_NODE) {
	elem := reflect.ValueOf(e).Elem()
	if !elem.IsValid() || !elem.CanSet() {
		log.Println("invalid struct pointer")
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
		// Unsupported field type
		default:
			log.Printf("Warning: unsupported field type for %s: %s", value.ValueName(), field.Kind())
		}
	}
	// Set LastWriteTime from key timestamp
	lastWriteTime := elem.FieldByName("LastWriteTime")
	if lastWriteTime.IsValid() && lastWriteTime.CanSet() {
		lastWriteTime.SetInt(key.LastWriteTime().Unix())
	}
}

type AmcacheTable string
const (
	ApplicationTable            AmcacheTable = "amcache_application"
	ApplicationFileTable        AmcacheTable = "amcache_application_file"
	ApplicationShortcutTable    AmcacheTable = "amcache_application_shortcut"
	DriverBinaryTable           AmcacheTable = "amcache_driver_binary"
	DevicePnpTable              AmcacheTable = "amcache_device_pnp"
)

// AllAmcacheTables returns a slice of all defined AmcacheTables.
func AllAmcacheTables() []AmcacheTable {
	return []AmcacheTable{
		ApplicationTable,
		ApplicationFileTable,
		ApplicationShortcutTable,
		DriverBinaryTable,
		DevicePnpTable,
	}
}

func (t AmcacheTable) Name() string {
	return string(t)
}

// GetHiveKey returns the registry hive key path associated with the AmcacheTable.
func (t AmcacheTable) GetHiveKey() string {
	switch t {
	case ApplicationTable:
		return "Root\\InventoryApplication"
	case ApplicationFileTable:
		return "Root\\InventoryApplicationFile"
	case ApplicationShortcutTable:
		return "Root\\InventoryApplicationShortcut"
	case DriverBinaryTable:
		return "Root\\InventoryDriverBinary"
	case DevicePnpTable:
		return "Root\\InventoryDevicePnp"
	default:
		return ""
	}
}

func (t AmcacheTable) NewEntry() Entry {
	switch t {
	case ApplicationTable:
		return &ApplicationEntry{}
	case ApplicationFileTable:
		return &ApplicationFileEntry{}
	case ApplicationShortcutTable:
		return &ApplicationShortcutEntry{}
	case DriverBinaryTable:
		return &DriverBinaryEntry{}
	case DevicePnpTable:
		return &DevicePnpEntry{}
	default:
		return nil
	}
}

func (t AmcacheTable) Columns() []table.ColumnDefinition {
	switch t {
	case ApplicationTable:
		return ApplicationColumns()
	case ApplicationFileTable:
		return ApplicationFileColumns()
	case ApplicationShortcutTable:
		return ApplicationShortcutColumns()
	case DriverBinaryTable:
		return DriverBinaryColumns()
	case DevicePnpTable:
		return DevicePnpColumns()
	default:
		return nil
	}
}

func (t AmcacheTable) GenerateFunc(state GlobalStateInterface, log *logger.Logger) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		filters := filters.GetConstraintFilters(queryContext)
		entries := state.GetCachedEntries(t, filters)
		marshalled, err := MarshalEntries(entries)
		if err != nil {
			return nil, err
		}
		return marshalled, nil
	}
}

// Registry reads the registry hive file and returns a regparser.Registry object.
func LoadRegistry(filePath string) (*regparser.Registry, error) {
	// ensure a path was provided
	if filePath == "" {
		return nil, fmt.Errorf("hive file path is empty")
	}

	// try reading the file directly first, which is faster if it works
	content, err := os.ReadFile(filePath)
	if err == nil {
		registry, err := regparser.NewRegistry(bytes.NewReader(content))
		if err == nil {
			return registry, nil
		}
	}

	// fallback to a low level read using fslib
	sourceFS, err := systemfs.New()
	if err != nil {
		return nil, err
	}
	fsPath, err := fslib.ToFSPath(filePath)
	if err != nil {
		return nil, err
	}
	content, err = fs.ReadFile(sourceFS, fsPath)
	if err != nil {
		return nil, err
	}

	registry, err := regparser.NewRegistry(bytes.NewReader(content))
	if err != nil {
		return nil, err
	}

	return registry, nil
}
