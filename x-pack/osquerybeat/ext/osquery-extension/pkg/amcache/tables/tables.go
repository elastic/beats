// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables

import (
	"log"
	"encoding/json"
	"bytes"
	"fmt"
	"github.com/forensicanalysis/fslib"
	"github.com/forensicanalysis/fslib/systemfs"
	"github.com/osquery/osquery-go/plugin/table"
	"io/fs"
	"os"
	"strings"
	"www.velocidex.com/golang/regparser"
)

const applicationKeyPath = "Root\\InventoryApplication"
const applicationFileKeyPath = "Root\\InventoryApplicationFile"
const applicationShortcutKeyPath = "Root\\InventoryApplicationShortcut"
const driverBinaryKeyPath = "Root\\InventoryDriverBinary"
const devicePnpKeyPath = "Root\\InventoryDevicePnp"

// GlobalState is an interface that defines methods for accessing global Amcache state.
type GlobalStateInterface interface {
	GetApplicationEntries(...string) []Entry
	GetApplicationFileEntries(...string) []Entry
	GetApplicationShortcutEntries(...string) []Entry
	GetDriverBinaryEntries(...string) []Entry
	GetDevicePnpEntries(...string) []Entry
}

// Entry is an interface that all Amcache table entry structs must implement.
// it is basically a row in a table.
type Entry interface {
	// SetLastWriteTime sets the last write time for the entry.
	SetLastWriteTime(int64)

	// FieldMappings returns a map of registry value names to struct field pointers for populating the entry.
	FieldMappings() map[string]*string
}

// FillInEntryFromKey takes an Entry, and using the FieldMappings, populates its fields from a registry key.
func FillInEntryFromKey(e Entry, key *regparser.CM_KEY_NODE) {
	// The regparser.CM_KEY_NODE has a Values() method that returns a slice of Value structs
	// Each Value struct has a ValueName() and ValueData() method but are not indexed in a map
	// so we create a map here for easy lookup
	subkeyMap := make(map[string]*regparser.ValueData)
	for _, value := range key.Values() {
		subkeyMap[value.ValueName()] = value.ValueData()
	}
	// Set FirstRunTime from key timestamp
	e.SetLastWriteTime(key.LastWriteTime().Unix())
	// Populate all fields using the mapping
	for registryKey, fieldPtr := range e.FieldMappings() {
		vd, ok := subkeyMap[registryKey]
		if !ok || vd == nil {
			// Not all fields are present in every entry, so just set to empty string
			*fieldPtr = ""
			continue
		}
		*fieldPtr = MakeTrimmedString(vd)
	}
}

// ToJson converts an Entry to its JSON string representation.
func ToJson(e Entry) string {
	j, err := json.Marshal(e)
	if err != nil {
		log.Printf("Error marshalling Entry to JSON: %v", err)
		return ""
	}
	return string(j)
}

// RowsAsStringMapArray converts a slice of Entry objects to a slice of maps with string keys and values.
// This is the format osquery expects for table rows.
func RowsAsStringMapArray(entries []Entry) []map[string]string {
	res := make([]map[string]string, 0, len(entries))
	for _, entry := range entries {
		j, err := json.Marshal(entry)
		if err != nil {
			log.Printf("Error marshalling Entry to JSON: %v", err)
			return nil
		}
		row := make(map[string]string)
		err = json.Unmarshal(j, &row)
		if err != nil {
			log.Printf("Error unmarshalling Entry JSON to map: %v", err)
			return nil
		}
		res = append(res, row)
	}
	return res
}

// MakeTrimmedString trims null characters from the end of a regparser ValueData string
// and returns it as a regular Go string. If the ValueData is a number, it returns its string representation.
// Otherwise, it returns a hex representation of the raw data.
func MakeTrimmedString(vd *regparser.ValueData) string {
	// regparser strings are null-terminated, and will show the trailing
	// nulls when marshalled to JSON. Trim them here.
	// Also, warn if the type is unexpected.

	if vd == nil {
		return ""
	}

	switch vd.Type {
	case regparser.REG_SZ, regparser.REG_EXPAND_SZ:
		return strings.TrimRight(vd.String, "\x00")
	case regparser.REG_DWORD, regparser.REG_QWORD:
		return fmt.Sprintf("%d", vd.Uint64)
	default:
		log.Printf("Warning: unexpected type for ValueData: %d", vd.Type)
		return fmt.Sprintf("%x", vd.Data)
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
				log.Printf("%s Query constraint: %d %s", fieldName, c.Operator, c.Expression)
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
