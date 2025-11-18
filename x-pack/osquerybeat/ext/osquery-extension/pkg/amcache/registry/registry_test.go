// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package registry

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

func TestRecovery(t *testing.T) {
	log := logger.New(os.Stdout, true)
	_, recovered, err := LoadRegistry("../testdata/recovery_data/Amcache.hve", log)
	assert.NoError(t, err, "failed to load registry")
	assert.False(t, recovered, "registry was not recovered unexpectedly")

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
