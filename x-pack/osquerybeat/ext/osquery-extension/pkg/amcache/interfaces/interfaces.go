// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

// Package interfaces defines interfaces for Amcache table entries and tables.
package interfaces

import (
	"encoding/json"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/utilities"
	"log"
	"www.velocidex.com/golang/regparser"
)

// Entry is an interface that all Amcache table entry structs must implement.
// it is basically a row in a table.
type Entry interface {
	// SetLastWriteTime sets the last write time for the entry.
	SetLastWriteTime(int64)

	// FieldMappings returns a map of registry value names to struct field pointers for populating the entry.
	FieldMappings() map[string]*string
}

// GlobalState is an interface that defines methods for accessing global Amcache state.
type GlobalState interface {
	GetApplicationEntries(...string) []Entry
	GetApplicationFileEntries(...string) []Entry
	GetApplicationShortcutEntries(...string) []Entry
	GetDriverBinaryEntries(...string) []Entry
	GetDevicePnpEntries(...string) []Entry
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
		*fieldPtr = utilities.MakeTrimmedString(vd)
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
