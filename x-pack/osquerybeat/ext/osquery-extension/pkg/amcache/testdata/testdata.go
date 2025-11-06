// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package testdata

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// MustGetRecoveryTestDataPath returns the path to the recovery test hive
func MustGetRecoveryTestDataPath(t *testing.T) string {
	absPath := filepath.Join(MustGetTestDataDirectory(t), "recovery_data", "Amcache.hve")
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		log.Printf("error: test hive path does not exist: %v", absPath)
		t.Fatal(err)
	}
	return absPath
}

// MustGetTestHivePath returns the path to the test hive
func MustGetTestHivePath(t *testing.T) string {
	absPath := filepath.Join(MustGetTestDataDirectory(t), "amcache.hve")
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		log.Printf("error: test hive path does not exist: %v", absPath)
		t.Fatal(err)
	}
	return absPath
}

// MustGetTestDataDirectory returns the path to the test data directory
func MustGetTestDataDirectory(t *testing.T) string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get current file path")
	}
	dir := filepath.Dir(currentFile)
	return dir
}
