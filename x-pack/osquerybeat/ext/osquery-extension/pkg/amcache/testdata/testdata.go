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

const TestHiveRelPath = "amcache.hve"

func GetRecoveryTestDataPathOrFatal(t *testing.T) string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get current file path")
	}
	dir := filepath.Dir(currentFile)
	absPath := filepath.Join(dir, "recovery_data")
	hivePath := filepath.Join(absPath, "Amcache.hve")
	if _, err := os.Stat(hivePath); os.IsNotExist(err) {
		log.Printf("error: test hive path does not exist: %v", hivePath)
		t.Fatal(err)
	}
	return hivePath
}

func GetTestHivePathOrFatal(t *testing.T) string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get current file path")
	}

	dir := filepath.Dir(currentFile)
	absPath := filepath.Join(dir, TestHiveRelPath)

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		log.Printf("error: test hive path does not exist: %v", absPath)
		t.Fatal(err)
	}
	return absPath
}
