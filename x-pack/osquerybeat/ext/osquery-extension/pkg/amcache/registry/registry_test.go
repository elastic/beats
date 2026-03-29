// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package registry

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	"www.velocidex.com/golang/regparser"
)

func TestRecovery(t *testing.T) {
	log := logger.New(os.Stdout, true)
	_, recovered, err := LoadRegistry("../testdata/recovery_data/Amcache.hve", log)
	assert.NoError(t, err, "failed to load registry")
	assert.True(t, recovered, "registry was not recovered unexpectedly")

	_, recovered, err = LoadRegistry("../testdata/Amcache.hve", log)
	assert.NoError(t, err, "failed to load registry")
	assert.False(t, recovered, "registry was recovered unexpectedly")
}

func Test_findTransactionLogs(t *testing.T) {
	log := logger.New(os.Stdout, true)

	type args struct {
		filePath string
		log      *logger.Logger
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		// recovery test data has 2 transaction logs
		{name: "recovery test data", args: args{filePath: "../testdata/recovery_data/Amcache.hve", log: log}, want: 2},

		// regular test data has no transaction logs
		{name: "regular test data", args: args{filePath: "../testdata/Amcache.hve", log: log}, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := len(findTransactionLogs(tt.args.filePath, tt.args.log))
			assert.Equal(t, tt.want, got, "findTransactionLogs() = %v, want %v", got, tt.want)
		})
	}
}

func Test_readFileViaNTFS(t *testing.T) {
	log := logger.New(os.Stdout, true)

	// Read the actual amcache hive using the readFileViaNTFS function to ensure it can read the file and return valid registry data.
	amcachePath := "C:\\Windows\\AppCompat\\Programs\\Amcache.hve"

	// Read the file using
	data, err := readFileViaNTFS(amcachePath)
	assert.NoError(t, err, "readFileViaNTFS() failed: %v", err)
	assert.NotEmpty(t, data, "readFileViaNTFS() returned empty data")

	magic := []byte{0x72, 0x65, 0x67, 0x66} // "regf"
	log.Infof("readFileViaNTFS() returned data with magic: %x", data[:5])
	assert.True(t, bytes.HasPrefix(data, magic), "readFileViaNTFS() returned data with incorrect magic")

	registry, err := regparser.NewRegistry(bytes.NewReader(data))
	assert.NoError(t, err, "failed to create registry from NTFS data")
	assert.NotNil(t, registry, "registry is nil")

	keyNode := registry.OpenKey("Root\\InventoryApplication")
	assert.NotNil(t, keyNode, "failed to open key Root\\InventoryApplication")
}

func TestLoadRegistry(t *testing.T) {
	log := logger.New(os.Stdout, true)

	tests := []struct {
		name          string // description of this test case
		filePath      string
		wantRecovered bool
		wantErr       bool
	}{
		{
			name:          "recovery test data",
			filePath:      "../testdata/recovery_data/Amcache.hve",
			wantRecovered: true,
			wantErr:       false,
		},
		{
			name:          "regular test data",
			filePath:      "../testdata/Amcache.hve",
			wantRecovered: false,
			wantErr:       false,
		},
		{
			name:          "empty file path",
			filePath:      "",
			wantRecovered: false,
			wantErr:       true,
		},
		{
			name:          "invalid file path",
			filePath:      "invalid/path",
			wantRecovered: false,
			wantErr:       true,
		},
		{
			name:          "valid file not registry",
			filePath:      "../testdata/not_a_registry.txt",
			wantRecovered: false,
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry, gotRecovered, err := LoadRegistry(tt.filePath, log)
			gotErr := err != nil

			assert.Equal(t, tt.wantErr, gotErr, "LoadRegistry() failed: errorExpected: %v, gotErr: %v", tt.wantErr, err)

			assert.Equal(t, tt.wantRecovered, gotRecovered, "LoadRegistry() recovered = %v, want %v", gotRecovered, tt.wantRecovered)

			if !tt.wantErr {
				assert.NotNil(t, registry, "LoadRegistry() registry is nil, want not nil")
			}
		})
	}
}
