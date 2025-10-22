// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
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
)

// TableType represents the different types of Amcache tables.
type TableType int
const (
	ApplicationTableType         TableType = iota // 0
	ApplicationFileTableType                      // 1
	ApplicationShortcutTableType                  // 2
	DriverBinaryTableType                         // 3
	DevicePnpTableType                            // 4
)

// AllTableTypes returns a slice of all defined TableTypes.
func AllTableTypes() []TableType {
	return []TableType{
		ApplicationTableType,
		ApplicationFileTableType,
		ApplicationShortcutTableType,
		DriverBinaryTableType,
		DevicePnpTableType,
	}
}

// GetHiveKey returns the registry hive key path associated with the TableType.
func (tt TableType) GetHiveKey() string {
	switch tt {
	case ApplicationTableType:
		return "Root\\InventoryApplication"
	case ApplicationFileTableType:
		return "Root\\InventoryApplicationFile"
	case ApplicationShortcutTableType:
		return "Root\\InventoryApplicationShortcut"
	case DriverBinaryTableType:
		return "Root\\InventoryDriverBinary"
	case DevicePnpTableType:
		return "Root\\InventoryDevicePnp"
	default:
		return ""
	}
}

// TableInterface defines the methods that each Amcache table must implement.
type TableInterface interface {
	Type() TableType
	Columns() []table.ColumnDefinition
	GenerateFunc(state GlobalStateInterface) table.GenerateFunc
	FilterColumn() string
}

// GlobalState is an interface that defines methods for accessing global Amcache state.
type GlobalStateInterface interface {
	GetCachedEntries(tableType TableType, ids ...string) []Entry
}

// Entry defines the methods that each Amcache entry must implement.
type Entry interface {
	FilterValue() string
	ToMap() (map[string]string, error)
}

// EntryFactory creates a new Entry instance based on the provided TableType.
func EntryFactory(tableType TableType) Entry {
	switch tableType {
	case ApplicationTableType:
		return &ApplicationEntry{}
	case ApplicationFileTableType:
		return &ApplicationFileEntry{}
	case ApplicationShortcutTableType:
		return &ApplicationShortcutEntry{}
	case DriverBinaryTableType:
		return &DriverBinaryEntry{}
	case DevicePnpTableType:
		return &DevicePnpEntry{}
	default:
		return nil
	}
}

// MarshalEntries takes a slice of Entry interfaces and marshals each to a map[string]string.
func MarshalEntries(entries []Entry) ([]map[string]string, error) {
	marshalled := make([]map[string]string, 0, len(entries))
	for _, entry := range entries {
		mapped, err := entry.ToMap()
		if err != nil {
			return nil, err
		}
		marshalled = append(marshalled, mapped)
	}
	return marshalled, nil
}

// GetEntriesFromRegistry reads the registry and returns a map of entries for the specified TableType.
func GetEntriesFromRegistry(tableType TableType, registry *regparser.Registry) (map[string][]Entry, error) {
	if registry == nil {
		return nil, fmt.Errorf("registry is nil")
	}

	keyNode := registry.OpenKey(tableType.GetHiveKey())
	if keyNode == nil {
		return nil, fmt.Errorf("error opening key: %s", tableType.GetHiveKey())
	}

	applicationEntries := make(map[string][]Entry, len(keyNode.Subkeys()))
	for _, subkey := range keyNode.Subkeys() {
		ae := EntryFactory(tableType)
		FillInEntryFromKey(ae, subkey)
		applicationEntries[ae.FilterValue()] = append(applicationEntries[ae.FilterValue()], ae)
	}
	return applicationEntries, nil
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

// GetConstraintsFromQueryContext extracts the constraints for a given field from the osquery QueryContext.
// It returns a slice of strings representing the constraint values.
// It only supports the '=' operator, and only for the indexed field (program_id, driver_id)
func GetConstraintsFromQueryContext(fieldName string, context table.QueryContext) []string {
	constraints := make([]string, 0)
	for name, cList := range context.Constraints {
		if len(cList.Constraints) > 0 && name == fieldName {
			for _, c := range cList.Constraints {
				if c.Operator != table.OperatorEquals {
					log.Printf("Warning: only '=' operator is supported for %s constraints, skipping %d", fieldName, c.Operator)
					continue
				}
				constraints = append(constraints, c.Expression)
			}
		}
	}
	return constraints
}
