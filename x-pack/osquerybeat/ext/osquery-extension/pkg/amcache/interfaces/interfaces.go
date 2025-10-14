// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

// Package interfaces defines interfaces for Amcache table entries and tables.
package interfaces

import (
	"encoding/json"
	"fmt"
	"log"
	"context"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/utilities"
	"github.com/osquery/osquery-go/plugin/table"
	"www.velocidex.com/golang/regparser"
)

// HiveReader defines an interface for reading a registry hive file.
// This is an interface to avoid circular dependencies.
type HiveReader interface {
	Registry() (*regparser.Registry, error)
}

// Entry represents a single entry in an Amcache table, such as an ApplicationFileEntry.
// Each table will have its own struct implementing this interface.
type Entry interface {
	// SetLastWriteTime sets the last write time for the entry.
	SetLastWriteTime(int64)

	// FieldMappings returns a map of registry value names to struct field pointers for populating the entry.
	FieldMappings() map[string]*string
}

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
		*fieldPtr = utilities.MakeTrimmedString(vd)
	}
}

// Table represents an Amcache table, such as ApplicationFileTable.
type Table interface {
	// Rows returns all entries in the table.
	Rows() []Entry

	// AddRow adds a new entry to the table from a registry key.
	AddRow(key *regparser.CM_KEY_NODE) error

	// KeyName returns the name of the registry key for the table.
	KeyName() string
}

func RowsAsStringMapArray(t Table) []map[string]string {
	res := make([]map[string]string, 0, len(t.Rows()))
	for _, entry := range t.Rows() {

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

func BuildTableFromRegistry(t Table, hiveReader HiveReader, ctx context.Context, queryContext table.QueryContext) error {
	registry, err := hiveReader.Registry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	keyNode := registry.OpenKey(t.KeyName())
	if keyNode == nil {
		return fmt.Errorf("failed to open key: %s", t.KeyName())
	}

	// for column_name, constraint_list := range queryContext.Constraints {
	// 	for _, constraint := range constraint_list.Constraints {
	// 		log.Printf("%s Query constraint: %d %s", column_name, constraint.Operator, constraint.Expression)
	// 	}
	// }

	for _, subkey := range keyNode.Subkeys() {
		t.AddRow(subkey)
	}
	return nil
}
