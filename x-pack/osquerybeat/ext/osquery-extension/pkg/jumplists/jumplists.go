// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

//go:generate go run ./generate

package jumplists

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/encoding"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	"github.com/osquery/osquery-go/plugin/table"
)

type JumpListType string

const (
	JumpListTypeCustom JumpListType = "custom"
)

type ApplicationId struct {
	Id string `osquery:"application_id"`
	Name string `osquery:"application_name"`
}

type JumpListMeta struct {
	ApplicationId
	jump_list_type   JumpListType `osquery:"jump_list_type"`
	path             string `osquery:"path"`
}

type JumpList struct {
	JumpListMeta
	lnks             []*Lnk
}

type JumpListRow struct {
	*JumpListMeta
	*Lnk
}

func (j *JumpList) ToRows() []JumpListRow {
	var rows []JumpListRow
	for _, lnk := range j.lnks {
		rows = append(rows, JumpListRow{
			JumpListMeta: &j.JumpListMeta,
			Lnk: lnk,
		})
	}
	if len(rows) == 0 {
		return []JumpListRow{
			{
				JumpListMeta: &j.JumpListMeta,
				Lnk: &Lnk{},
			},
		}
	}
	return rows
}

func NewApplicationId(id string) ApplicationId {
	name, ok := jumpListAppIds[id]; if !ok {
		name = ""
	}
	return ApplicationId{
		Id: id,
		Name: name,
	}
}

func NewApplicationIdFromFileName(fileName string, log *logger.Logger) ApplicationId {
	baseName  := filepath.Base(fileName)
	dotIndex := strings.Index(baseName, ".")
	if dotIndex != -1 {
		return NewApplicationId(baseName[:dotIndex])
	}

	// Not necessarily an error, just a fallback
	log.Infof("failed to get application id from file name %s", fileName)
	return ApplicationId{}
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

// GenerateFunc
func GetGenerateFunc(log *logger.Logger) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		jumpLists := GetCustomJumpLists(log)

		var rows []map[string]string
		for _, jumpList := range jumpLists {
			for _, row := range jumpList.ToRows() {
				rowMap, err := encoding.MarshalToMap(row)
				if err != nil {
					return nil, err
				}
				rows = append(rows, rowMap)
			}
		}
		return rows, nil
	}
}
