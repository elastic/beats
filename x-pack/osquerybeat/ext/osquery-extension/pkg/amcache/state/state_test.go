// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package state

import (
	"log"
	"os"
	"path/filepath"
	"testing"
)

func GetTestHivePath() (string, error) {
	relPath := filepath.Join("..", "testdata", "Amcache.hve")
	absPath, err := filepath.Abs(relPath)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return "", err
	}
	return absPath, nil
}

func TestGlobalState(t *testing.T) {
	testHivePath, err := GetTestHivePath()
	if err != nil {
		t.Fatal("Failed to get test hive path")
	}
	SetHivePath(testHivePath)

	instance := GetInstance()
	if instance == nil {
		t.Fatal("Expected instance to be initialized")
	}

	applicationEntries := instance.GetApplicationEntries()
	log.Printf("Application entries loaded: %d", len(applicationEntries))
}
