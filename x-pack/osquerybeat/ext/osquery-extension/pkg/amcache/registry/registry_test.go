// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

func getTestDataDirectory() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get current file path")
	}
	dir := filepath.Dir(currentFile)
	testDataDirectory := filepath.Join(dir, "..", "testdata")
	if _, err := os.Stat(testDataDirectory); os.IsNotExist(err) {
		return "", fmt.Errorf("test data directory does not exist: %w", err)
	}
	return testDataDirectory, nil
}

func getTestHivePath() (string, error) {
	testDataDirectory, err := getTestDataDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(testDataDirectory, "amcache.hve"), nil
}

func getRecoveryTestDataPath() (string, error) {
	testDataDirectory, err := getTestDataDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(testDataDirectory, "recovery_data", "Amcache.hve"), nil
}

func TestRecovery(t *testing.T) {
	log := logger.New(os.Stdout, true)
	filePath, err := getRecoveryTestDataPath()
	if err != nil {
		t.Fatalf("failed to get recovery test data path: %v", err)
	}
	_, recovered, err := LoadRegistry(filePath, log)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}
	if !recovered {
		t.Fatalf("registry was not recovered unexpectedly")
	}

	filePath, err = getTestHivePath()
	if err != nil {
		t.Fatalf("failed to get test hive path: %v", err)
	}
	_, recovered, err = LoadRegistry(filePath, log)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}
	if recovered {
		t.Fatalf("registry was recovered unexpectedly")
	}
}

func Test_findTransactionLogs(t *testing.T) {
	recoveryTestDataPath, err := getRecoveryTestDataPath()
	if err != nil {
		t.Fatalf("failed to get recovery test data path: %v", err)
	}
	regularTestDataPath, err := getTestHivePath()
	if err != nil {
		t.Fatalf("failed to get test hive path: %v", err)
	}
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

	recoveryTestDataPath, err := getRecoveryTestDataPath()
	if err != nil {
		t.Fatalf("failed to get recovery test data path: %v", err)
	}
	regularTestDataPath, err := getTestHivePath()
	if err != nil {
		t.Fatalf("failed to get test hive path: %v", err)
	}
	testDataDirectory, err := getTestDataDirectory()
	if err != nil {
		t.Fatalf("failed to get test data directory: %v", err)
	}
	notARegistryPath := filepath.Join(testDataDirectory, "not_a_registry.txt")
	tests := []struct {
		name          string // description of this test case
		filePath      string
		wantRecovered bool
		wantErr       bool
	}{
		{
			name:          "recovery test data",
			filePath:      recoveryTestDataPath,
			wantRecovered: true,
			wantErr:       false,
		},
		{
			name:          "regular test data",
			filePath:      regularTestDataPath,
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
			filePath:      notARegistryPath,
			wantRecovered: false,
			wantErr:       true,
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
