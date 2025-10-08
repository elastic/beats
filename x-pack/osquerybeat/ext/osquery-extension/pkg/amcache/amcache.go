// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

// Package amcache provides functions to read and parse the Windows Amcache.hve registry hive file.
package amcache

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"

	"github.com/forensicanalysis/fslib/systemfs"
	"github.com/osquery/osquery-go/plugin/table"
	"www.velocidex.com/golang/regparser"
)

// InventoryApplicationFileEntry represents a single entry in the InventoryApplicationFile
// key of the Amcache.hve
type InventoryApplicationFileEntry struct {
	Name              string `json:"name"`
	FirstRunTime      string `json:"first_run_time"`
	ProgramId         string `json:"program_id"`
	FileId            string `json:"file_id"`
	LowerCaseLongPath string `json:"lower_case_long_path"`
	OriginalFileName  string `json:"original_file_name"`
	Publisher         string `json:"publisher"`
	Version           string `json:"version"`
	BinFileVersion    string `json:"bin_file_version"`
	BinaryType        string `json:"binary_type"`
	ProductName       string `json:"product_name"`
	ProductVersion    string `json:"product_version"`
	LinkDate          string `json:"link_date"`
	BinProductVersion string `json:"bin_product_version"`
	Size              string `json:"size"`
	Language          string `json:"language"`
	Usn               string `json:"usn"`
}

// AsStringMap converts the InventoryApplicationFileEntry to a map[string]string
// suitable for returning as a row in an osquery table.
func (iafe *InventoryApplicationFileEntry) AsStringMap() map[string]string {
	j, err := json.Marshal(iafe)
	if err != nil {
		log.Printf("Error marshalling InventoryApplicationFileEntry to JSON: %v", err)
		return nil
	}
	row := make(map[string]string)
	err = json.Unmarshal(j, &row)
	if err != nil {
		log.Printf("Error unmarshalling InventoryApplicationFileEntry JSON to map: %v", err)
		return nil
	}
	return row
}

// FillInFromKey populates the InventoryApplicationFileEntry fields from a regparser.CM_KEY_NODE
// representing a subkey of the InventoryApplicationFile key.
func (iafe *InventoryApplicationFileEntry) FillInFromKey(key *regparser.CM_KEY_NODE) {

	// Define field mappings to eliminate repetitive code
	fieldMappings := map[string]*string{
		"Name":              &iafe.Name,
		"ProgramId":         &iafe.ProgramId,
		"FileId":            &iafe.FileId,
		"LowerCaseLongPath": &iafe.LowerCaseLongPath,
		"OriginalFileName":  &iafe.OriginalFileName,
		"Publisher":         &iafe.Publisher,
		"Version":           &iafe.Version,
		"BinFileVersion":    &iafe.BinFileVersion,
		"BinaryType":        &iafe.BinaryType,
		"ProductName":       &iafe.ProductName,
		"ProductVersion":    &iafe.ProductVersion,
		"LinkDate":          &iafe.LinkDate,
		"BinProductVersion": &iafe.BinProductVersion,
		"Size":              &iafe.Size,
		"Language":          &iafe.Language,
		"Usn":               &iafe.Usn,
	}

	// The regparser.CM_KEY_NODE has a Values() method that returns a slice of Value structs
	// Each Value struct has a ValueName() and ValueData() method but are not indexed in a map
	// so we create a map here for easy lookup
	subkeyMap := make(map[string]*regparser.ValueData)
	for _, value := range key.Values() {
		subkeyMap[value.ValueName()] = value.ValueData()
	}

	// Set FirstRunTime from key timestamp
	iafe.FirstRunTime = fmt.Sprintf("%d", key.LastWriteTime().Unix())

	// Populate all fields using the mapping
	for registryKey, fieldPtr := range fieldMappings {
		vd, ok := subkeyMap[registryKey]
		if !ok || vd == nil {
			// Not all fields are present in every entry, so just set to empty string
			*fieldPtr = ""
			continue
		}
		*fieldPtr = MakeTrimmedString(vd)
	}
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


// ReadAmcacheHiveBytes reads the Amcache.hve file from the given file path.
// If no file path is provided, it reads the hive from the live filesystem.
// It returns the raw bytes of the hive file.
func ReadAmcacheHiveBytes(filePath ...string) ([]byte, error) {
	if len(filePath) > 1 {
		return nil, fmt.Errorf("only one file path argument is allowed")
	}

	// Only read normally if a non-empty file path is provided
	if len(filePath) == 1  && len(filePath[0]) > 0 {
		log.Printf("Reading Amcache hive from file: %s", filePath[0])
		return os.ReadFile(filePath[0])
	}

	// If no file path is provided, read the hive from the live filesystem
	// Because this file is usually locked, we have to read it forensically
	// using the systemfs package
	log.Printf("Reading Amcache hive from live filesystem")

	sourceFS, err := systemfs.New()
	if err != nil {
		return nil, err
	}

	return fs.ReadFile(sourceFS, "C/Windows/AppCompat/Programs/Amcache.hve")
}

// GetAmcacheRegistry reads the Amcache.hve file and returns a regparser.Registry object.
// If a file path is provided, it reads the hive from that file.
// Otherwise, it reads the hive from the live filesystem.
func GetAmcacheRegistry(filePath ...string) (*regparser.Registry, error) {
	content, err := ReadAmcacheHiveBytes(filePath...)
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(content)
	registry, err := regparser.NewRegistry(reader)
	if err != nil {
		return nil, err
	}
	return registry, nil
}

// GetInventoryApplicationFileEntries retrieves all entries under the
// InventoryApplicationFile key in the Amcache.hve registry hive.
// If a file path is provided, it reads the hive from that file.
// Otherwise, it reads the hive from the live filesystem.
func GetInventoryApplicationFileEntries(filePath ...string) ([]InventoryApplicationFileEntry, error) {
	registry, err := GetAmcacheRegistry(filePath...)
	if err != nil {
		return nil, err
	}

	keyNode := registry.OpenKey("Root/InventoryApplicationFile")
	if keyNode == nil {
		return nil, fmt.Errorf("could not open key")
	}

	res := make([]InventoryApplicationFileEntry, 0, len(keyNode.Subkeys()))

	for _, subkey := range keyNode.Subkeys() {
		iaf := InventoryApplicationFileEntry{}
		iaf.FillInFromKey(subkey)
		res = append(res, iaf)
	}
	log.Printf("Found %d InventoryApplicationFile entries", len(res))

	return res, nil
}

// GenAmcacheTable generates the rows for the Amcache osquery table.
// It reads the Amcache.hve from the provided file path, or from the live filesystem if no path is given.
func GenAmcacheTable(ctx context.Context, queryContext table.QueryContext, filePath ...string) ([]map[string]string, error) {
	// TODO: Use queryContext to filter results if needed
	res := make([]map[string]string, 0)
	entries, err := GetInventoryApplicationFileEntries(filePath...)
	if err != nil {
		return nil, fmt.Errorf("error getting InventoryApplicationFileEntries: %w", err)
	}
	for _, entry := range entries {
		res = append(res, entry.AsStringMap())
	}
	log.Printf("Generated %d rows for Amcache table", len(res))
	return res, nil
}
