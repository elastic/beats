// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one

// or more contributor license agreements. Licensed under the Elastic License;

// you may not use this file except in compliance with the Elastic License.

//go:build windows

package registry

import (
	"os"
	"testing"
	// "path/filepath"
	//"fmt"

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
	transactionLogPaths := findTransactionLogs(filePath, log)
	registry, err := recoverRegistry(filePath, transactionLogPaths, log)

	key :=registry.OpenKey("Root\\InventoryApplication")
	if key == nil {
		t.Fatalf("failed to open key: %v", err)
	}
	if len(key.Subkeys()) == 0 {
		t.Fatalf("key has no subkeys")
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
