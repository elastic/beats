// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package driver_binary

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/interfaces"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/utilities"
	"github.com/osquery/osquery-go/plugin/table"
)


func TestDriverBinaryTable(t *testing.T) {
    testCases := []struct {
        name     string
        filePath string
        table    interfaces.Table
    }{
        {name: "Gen table from cached file", filePath: "..\\..\\testdata\\Amcache.hve", table: &DriverBinaryTable{}},
        {name: "Gen table from live system", filePath: "C:\\Windows\\AppCompat\\Programs\\Amcache.hve", table: &DriverBinaryTable{}},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            if tc.filePath != "" {
                if _, err := os.Stat(tc.filePath); os.IsNotExist(err) {
                    t.Fatalf("File does not exist: %s", tc.filePath)
                }
            }
            hiveReader := utilities.HiveReader{FilePath: tc.filePath}

            ctx := context.Background()
			generateFunc := GenerateFunc(&hiveReader)
            table, err := generateFunc(ctx, table.QueryContext{})
            if err != nil {
                t.Errorf("BuildTableFromRegistry() failed: %v", err)
            }

            for _, row := range table {
				log.Println(row)
                for _, column := range DriverBinaryColumns() {
                    if _, ok := row[column.Name]; !ok {
                        t.Errorf("Missing expected key '%s' in row: %v", column.Name, row)
                    }
                }
            }
        })
    }
}
