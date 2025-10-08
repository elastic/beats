//go:build windows
package amcache

import (
    "testing"
    "os"
    "github.com/osquery/osquery-go/plugin/table"
    "context"
)

var ExpectedKeys = []string{
    "name",
    "first_run_time",
    "program_id",
    "file_id",
    "lower_case_long_path",
    "original_file_name",
    "publisher",
    "version",
    "bin_file_version",
    "binary_type",
    "product_name",
    "product_version",
    "link_date",
    "bin_product_version",
    "size",
    "language",
    "usn",
}

func TestGenAmcacheTable(t *testing.T) {
    testCases := []struct {
        name     string
        filePath string
    }{
        {name: "Gen table from cached file", filePath: "testdata/Amcache.hve"},
        {name: "Gen table from live system", filePath: ""},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            if tc.filePath != "" {
                if _, err := os.Stat(tc.filePath); os.IsNotExist(err) {
                    t.Fatalf("File does not exist: %s", tc.filePath)
                }
            }
            table, err := GenAmcacheTable(context.Background(), table.QueryContext{}, tc.filePath)
            if err != nil {
                t.Errorf("ReadAmcacheHive() failed: %v", err)
            }

            for _, row := range table {
                for _, key := range ExpectedKeys {
                    if _, ok := row[key]; !ok {
                        t.Errorf("Missing expected key '%s' in row: %v", key, row)
                    }
                }
            }
        })
    }
}