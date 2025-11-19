// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package jumplists

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/encoding"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/parsers/resources"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	"github.com/osquery/osquery-go/plugin/table"
)

type JumpListType string

const (
	JumpListTypeCustom JumpListType = "custom"
)

type JumpList interface {
	Path() string
	AppId() resources.ApplicationId
	Type() JumpListType
	ToRows() []JumpListRow
}

type JumpListRow struct {
	Path               string    `osquery:"path"`
	ApplicationId      string    `osquery:"application_id"`
	ApplicationName    string    `osquery:"application_name"`
	JumpListType       string    `osquery:"jump_list_type"`
	TargetCreatedTime  time.Time `osquery:"target_created_time" format:"unix"`
	TargetModifiedTime time.Time `osquery:"target_modified_time"`
	TargetAccessedTime time.Time `osquery:"target_accessed_time"`
	TargetSize         uint32    `osquery:"target_size"`
	TargetPath         string    `osquery:"target_path"`
}

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

func GetColumns() []table.ColumnDefinition {
	columns, err := encoding.GenerateColumnDefinitions(JumpListRow{})
	if err != nil {
		return nil
	}
	return columns
}

// GenerateFunc generates the data for the ApplicationFileTable based on the provided GlobalStateInterface.
func GetGenerateFunc(log *logger.Logger) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		// jumpLists, err := GetCustomDestinationJumpLists(log)
		// if err != nil {
		// 	return nil, err
		// }
		var rows []map[string]string
		// for _, jumpList := range jumpLists {
		// 	for _, row := range jumpList.ToRows() {
		// 		rowMap, err := encoding.MarshalToMap(row)
		// 		if err != nil {
		// 			return nil, err
		// 		}
		// 		rows = append(rows, rowMap)
		// 	}
		// }
		return rows, nil
	}
}
