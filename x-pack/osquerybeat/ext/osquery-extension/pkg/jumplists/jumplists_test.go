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

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/encoding"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

func GetAllFilesInDirectory(dir string, t *testing.T) []string {
	dir, err := filepath.Abs(dir); if err != nil {
		t.Fatalf("GetAllFilesInDirectory() returned error: %v", err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "*")); if err != nil {
		t.Fatalf("GetAllFilesInDirectory() returned error: %v", err)
	}
	return files
}

func TestCustomJumplists(t *testing.T) {
	files := GetAllFilesInDirectory("./testdata/custom", t)

	log := logger.New(os.Stdout, true)
	for _, file := range files {
		jumplist, err := ParseCustomJumpListFile(file, log)
		assert.NoError(t, err, "expected no error when parsing custom jumplist")
		assert.NotNil(t, jumplist, "expected non-nil jumplist when parsing custom jumplist")

		rows := jumplist.ToRows()
		assert.GreaterOrEqual(t, len(rows), 1, "expected at least one row in the jumplist")

		for _, row := range rows {
			marshalled, err := encoding.MarshalToMap(row)
			assert.NoError(t, err, "expected no error when marshaling custom jumplist")
			assert.NotNil(t, marshalled, "expected non-nil marshalled row when marshaling custom jumplist")
		}
	}
}

func TestEmptyCustomJumplist(t *testing.T) {
	files := GetAllFilesInDirectory("./testdata/custom_empty", t)
	assert.GreaterOrEqual(t, len(files), 1, "expected at least one file in the empty custom jumplist directory")
	log := logger.New(os.Stdout, true)
	for _, file := range files {
		jumplist, err := ParseCustomJumpListFile(file, log)
		assert.Error(t, err, "expected error when parsing empty custom jumplist")
		assert.Nil(t, jumplist, "expected nil jumplist when parsing empty custom jumplist")
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
			fmt.Printf("Lnk: %v\n", got)
		})
	}
}