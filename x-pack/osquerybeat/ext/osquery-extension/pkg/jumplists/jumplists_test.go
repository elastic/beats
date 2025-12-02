// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package jumplists

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	// "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/encoding"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

func GetAllFilesInDirectory(dir string, t *testing.T) []string {
	dir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("GetAllFilesInDirectory() returned error: %v", err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		t.Fatalf("GetAllFilesInDirectory() returned error: %v", err)
	}
	return files
}

func TestCustomJumplists(t *testing.T) {

	type testCase struct {
		name         string
		filePath     string
		expectError  bool
		expectedRows int
	}

	tests := []testCase{
		{
			name:         "test_custom_jumplist_1",
			filePath:     "./testdata/custom/7e4dca80246863e3.customDestinations-ms",
			expectError:  true,
			expectedRows: 1,
		},
		{
			name:         "test_custom_jumplist_2",
			filePath:     "./testdata/custom/590aee7bdd69b59b.customDestinations-ms",
			expectError:  false,
			expectedRows: 3,
		},
		{
			name:         "test_custom_jumplist_3",
			filePath:     "./testdata/custom/ccba5a5986c77e43.customDestinations-ms",
			expectError:  false,
			expectedRows: 2,
		},
		{
			name:         "test_custom_jumplist_4",
			filePath:     "./testdata/custom/f4ed0c515fdbcbc.customDestinations-ms",
			expectError:  false,
			expectedRows: 3,
		},
		{
			name:         "test_custom_jumplist_5",
			filePath:     "./testdata/custom/f4ed0c515fdbcbc.customDestinations-ms",
			expectError:  false,
			expectedRows: 3,
		},
	}

	log := logger.New(os.Stdout, true)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			jumplist, err := ParseCustomJumpListFile(test.filePath, log)
			if test.expectError {
				assert.Error(t, err, "expected error when parsing custom jumplist")
				assert.Nil(t, jumplist, "expected nil jumplist when parsing custom jumplist")
				return
			}
			assert.NoError(t, err, "expected no error when parsing custom jumplist")
			assert.NotNil(t, jumplist, "expected non-nil jumplist when parsing custom jumplist")
			rows := jumplist.ToRows()
			assert.Equal(t, test.expectedRows, len(rows), "expected %d rows in the jumplist", test.expectedRows)
		})
	}
}

func TestLnkFromPath(t *testing.T) {
	type args struct {
		filePath string
	}
	tests := []struct {
		name    string
		args    args
		want    *Lnk
		wantErr bool
	}{
		{
			name: "test_lnk_36.bin",
			args: args{
				filePath: "./testdata/lnks/lnk_36.bin",
			},
		},
		{
			name: "test_lnk_48.bin",
			args: args{
				filePath: "./testdata/lnks/lnk_48.bin",
			},
		},
		{
			name: "test_lnk_1332.bin",
			args: args{
				filePath: "./testdata/lnks/lnk_1332.bin",
			},
		},
		{
			name: "test_lnk_5312.bin",
			args: args{
				filePath: "./testdata/lnks/lnk_5312.bin",
			},
		},
	}

	log := logger.New(os.Stdout, true)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewLnkFromPath(tt.args.filePath, log)
			if err != nil {
				t.Errorf("NewLnkFromPath() error = %v", err)
				return
			}
		})
	}
}
