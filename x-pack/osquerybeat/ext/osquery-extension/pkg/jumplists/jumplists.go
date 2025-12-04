// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

// generate the application id map
//go:generate go run ./generate

package jumplists

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/encoding"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

type JumpListType string

const (
	JumpListTypeCustom JumpListType = "custom"
	JumpListTypeAutomatic JumpListType = "automatic"
)

// JumpListMeta is the metadata for a jump list.
// It contains the application ID, jump list type, path to the jump list file,
// and any jumplist type specific metadata.
type JumpListMeta struct {
	ApplicationId
	JumplistType JumpListType `osquery:"jumplist_type"`
	Path         string       `osquery:"path"`
}

// JumpListEntry is a single entry in a jump list.
type JumpListEntry struct {
	*DestListEntry // Only used for automatic jumplists
	*Lnk
}

// JumpList is a collection of Lnk objects that represent a single jump list.
// It contains the metadata for the jump list and the Lnk objects.
// This is a generic object that can represent either a custom jumplist
// or an automatic jumplist.  The JumpListMeta contains any data specific to the jumplist type.
// Both jumplist types are comprised of a collection of Lnk objects.
type JumpList struct {
	JumpListMeta
	entries []*JumpListEntry
}

// JumpListRow is a single row in a jump list.
// Each lnk object in the jumplist is represented by its own row, so the number of rows
// is equal to the number of lnk objects in the jumplist.
type JumpListRow struct {
	*JumpListMeta  // The metadata for the jump list
	*JumpListEntry // The JumpListEntry object that represents a single jump list entry
}

// ToRows converts the JumpList to a slice of JumpListRow objects.
// If the JumpList is empty, it returns a single empty JumpListRow.
func (j *JumpList) ToRows() []JumpListRow {
	var rows []JumpListRow
	for _, entry := range j.entries {
		rows = append(rows, JumpListRow{
			JumpListMeta: &j.JumpListMeta,
			JumpListEntry: entry,
		})
	}
	// If the jumplist is empty, return a single empty JumpListRow. Which
	// will still have the metadata for the jump list (application id, jumplist type, path, etc)
	if len(rows) == 0 {
		return []JumpListRow{
			{
				JumpListMeta: &j.JumpListMeta,
				JumpListEntry: &JumpListEntry{
					DestListEntry: nil,
					Lnk:           nil,
				},
			},
		}
	}
	return rows
}

// NewApplicationIdFromFileName creates a new ApplicationId from a given file name.
// The file name is the name of the jumplist file.
// The application ID contains all .
// The application name is looked up from the jumpListAppIds map.
// If the application ID is not found, the name is set to an empty string.
func NewApplicationIdFromFileName(fileName string, log *logger.Logger) ApplicationId {
	baseName := filepath.Base(fileName)
	dotIndex := strings.Index(baseName, ".")
	if dotIndex != -1 {
		return NewApplicationId(baseName[:dotIndex])
	}

	// Not necessarily an error, just a fallback
	log.Infof("failed to get application id from file name %s", fileName)
	return ApplicationId{}
}

// FindJumplistFiles finds all the jump list files of a given type.
// It returns a slice of file paths.
func FindJumplistFiles(jumplistType JumpListType, log *logger.Logger) ([]string, error) {
	// Get the path to the automatic jumplist directory
	var path string

	switch jumplistType {
	case JumpListTypeCustom:
		path = "$APPDATA\\Microsoft\\Windows\\Recent\\CustomDestinations"
	}

	expandedPath := os.ExpandEnv(path)
	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		return nil, err
	}

	// Get a list of the files in the directory
	fileEntries, err := os.ReadDir(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", expandedPath, err)
	}

	// Iterate over the file entries and build a list of file paths
	var files []string // Return a list of file paths
	for _, entry := range fileEntries {
		filePath := filepath.Join(expandedPath, entry.Name())
		files = append(files, filePath)
	}

	// Return the list of file paths (absolute paths)
	return files, nil
}

// GetColumns returns the column definitions for the JumpListRow object.
// It returns a slice of table.ColumnDefinition objects.
func GetColumns() []table.ColumnDefinition {
	columns, err := encoding.GenerateColumnDefinitions(JumpListRow{})
	if err != nil {
		return nil
	}
	return columns
}

// GetGenerateFunc returns a function that can be used to generate a table of JumpListRow objects.
// It returns a function that can be used to generate a table of JumpListRow objects.
func GetGenerateFunc(log *logger.Logger) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		jumpLists := GetCustomJumpLists(log)

		var rows []map[string]string
		for _, jumpList := range jumpLists {
			for _, row := range jumpList.ToRows() {
				rowMap, err := encoding.MarshalToMapWithFlags(row, encoding.EncodingFlagUseNumbersZeroValues)
				if err != nil {
					return nil, err
				}
				rows = append(rows, rowMap)
			}
		}
		return rows, nil
	}
}
