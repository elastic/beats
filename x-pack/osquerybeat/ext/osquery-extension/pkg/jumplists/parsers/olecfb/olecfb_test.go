// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one

// or more contributor license agreements. Licensed under the Elastic License;

// you may not use this file except in compliance with the Elastic License.

//go:build windows

package olecfb

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	//"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/parsers/lnk"
)

func MustGetAutomaticDestinationFiles(t *testing.T) []string {
	_, currentFile, _, ok := runtime.Caller(0)
	currentDirectory := filepath.Dir(currentFile)
	automaticDestinationsDirectory := filepath.Join(currentDirectory, "..", "..", "testdata", "automatic_destinations")
	if !ok {
		t.Fatal("failed to get current file path")
	}
	fmt.Printf("automaticDestinationsDirectory: %s\n", automaticDestinationsDirectory)
	if _, err := os.Stat(automaticDestinationsDirectory); os.IsNotExist(err) {
		t.Fatal("automatic destinations directory does not exist")
	}

	// Get a list of the files in the directory
	fileEntries, err := os.ReadDir(automaticDestinationsDirectory)
	if err != nil {
		t.Fatalf("failed to read directory %s: %v", automaticDestinationsDirectory, err)
	}
	files := []string{}
	for _, file := range fileEntries {
		files = append(files, filepath.Join(automaticDestinationsDirectory, file.Name()))
	}
	return files
}

func TestNewOlecfb(t *testing.T) {
	log := logger.New(os.Stdout, true)
	files := MustGetAutomaticDestinationFiles(t)
	if len(files) == 0 {
		t.Errorf("No automatic destination files found")
	}
	parsedFiles := []*Olecfb{}
	for _, file := range files {
		olecfb, err := NewOlecfb(file, log)
		if err != nil {
			t.Fatalf("NewOlecfb() returned error: %v", err)
		}
		if !olecfb.HasValidDestList() {
			log.Infof("Olecfb is empty: %s", file)
			continue
		}
		parsedFiles = append(parsedFiles, olecfb)
	}
	if len(parsedFiles) == 0 {
		t.Errorf("No parsed files found")
	}
	for _, olecfb := range parsedFiles {
		if int(olecfb.DestList.Header.NumberOfEntries) != len(olecfb.DestList.Entries) {
			t.Errorf("NumberOfEntries and Entries length mismatch")
		}
	}
}
