// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package registry

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/testdata"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

// func TestLoadRegistry(t *testing.T) {
// 	filePath := testdata.GetTestHivePathOrFatal(t)
// 	registry, err := LoadRegistry(filePath)
// 	if err != nil {
// 		t.Fatalf("failed to load registry: %v", err)
// 	}
// 	if registry == nil {
// 		t.Fatalf("registry is nil")
// 	}
// 	if len(registry.Subkeys()) == 0 {
// 		t.Fatalf("registry has no subkeys")
// 	}
// 	if len(registry.Values()) == 0 {
// 		t.Fatalf("registry has no values")
// 	}
// }

func TestRecovery(t *testing.T) {
	log := logger.New(os.Stdout, true)
	filePath := testdata.GetRecoveryTestDataPathOrFatal(t)
	_, recovered, err := LoadRegistry(filePath, log)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}
	if !recovered {
		t.Fatalf("registry was not recovered unexpectedly")
	}

	filePath = testdata.GetTestHivePathOrFatal(t)
	_, recovered, err = LoadRegistry(filePath, log)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}
	if recovered {
		t.Fatalf("registry was recovered unexpectedly")
	}
}

// func Test_getFileContents(t *testing.T) {
// 	tests := []struct {
// 		name string // description of this test case
// 		// Named input parameters for target function.
// 		filePath string
// 		log      *logger.Logger
// 		want     []byte
// 		wantErr  bool
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got, gotErr := getFileContents(tt.filePath, tt.log)
// 			if gotErr != nil {
// 				if !tt.wantErr {
// 					t.Errorf("getFileContents() failed: %v", gotErr)
// 				}
// 				return
// 			}
// 			if tt.wantErr {
// 				t.Fatal("getFileContents() succeeded unexpectedly")
// 			}
// 			// TODO: update the condition below to compare got with tt.want.
// 			if true {
// 				t.Errorf("getFileContents() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

func Test_findTransactionLogs(t *testing.T) {

	recoveryTestDataPath := testdata.GetRecoveryTestDataPathOrFatal(t)
	regularTestDataPath := testdata.GetTestHivePathOrFatal(t)
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
		{name: "recovery test data", args: args{filePath: recoveryTestDataPath, log: log}, want: 2},

		// regular test data has no transaction logs
		{name: "regular test data", args: args{filePath: regularTestDataPath, log: log}, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := len(findTransactionLogs(tt.args.filePath, tt.args.log)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s: findTransactionLogs() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestLoadRegistry(t *testing.T) {

	log := logger.New(os.Stdout, true)

	tests := []struct {
		name string // description of this test case
		filePath string
		wantRecovered    bool
		wantErr      bool	
	}{
		//
		{
			name: "recovery test data", 
			filePath: testdata.GetRecoveryTestDataPathOrFatal(t),
			wantRecovered: true,
			wantErr: false,
		},
		{
			name: "regular test data", 
			filePath: testdata.GetTestHivePathOrFatal(t),
			wantRecovered: false,
			wantErr: false,
		},
		{
			name: "empty file path",
			filePath: "",
			wantRecovered: false,
			wantErr: true,
		},
		{
			name: "invalid file path",
			filePath: "invalid/path",
			wantRecovered: false,
			wantErr: true,
		},
		{
			name: "valid file not registry",
			filePath: filepath.Join(testdata.GetTestDataDirectoryOrFatal(t), "not_a_registry.txt"),
			wantRecovered: false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry, gotRecovered, err := LoadRegistry(tt.filePath, log)
            gotErr := err != nil

			if gotErr != tt.wantErr {
				t.Errorf("%s: LoadRegistry() failed: errorExpected: %v, gotErr: %v", tt.name, tt.wantErr, err)
				return
			}

			if gotRecovered != tt.wantRecovered {
				t.Errorf("%s: LoadRegistry() recovered = %v, want %v", tt.name, gotRecovered, tt.wantRecovered)
				return
			}

			if !tt.wantErr && registry == nil {
				t.Errorf("%s: LoadRegistry() registry is nil, want error", tt.name)
				return
			}
		})
	}
}
