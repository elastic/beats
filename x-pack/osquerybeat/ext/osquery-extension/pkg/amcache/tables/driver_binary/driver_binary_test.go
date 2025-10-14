// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package driver_binary

import (
	"os"
	"path/filepath"
	"testing"
)

func GetTestHivePath() (string, error) {
	relPath := filepath.Join("..", "..", "testdata", "Amcache.hve")
	absPath, err := filepath.Abs(relPath)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return "", err
	}
	return absPath, nil
}

func TestDriverBinary(t *testing.T) {
	test_hive_path, err := GetTestHivePath()
	if err != nil {
		t.Fatalf("Failed to get test hive path: %v", err)
	}
	if _, err := os.Stat(test_hive_path); os.IsNotExist(err) {
		t.Fatalf("File does not exist: %s", test_hive_path)
	}
}
