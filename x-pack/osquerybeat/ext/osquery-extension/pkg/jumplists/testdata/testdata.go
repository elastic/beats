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


func GetCurrentDirectoryOrFatal(t *testing.T) string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get current file path")
	}
	return filepath.Dir(currentFile)
}

func EnsureDirectoryExistsOrFatal(t *testing.T, directory string) {
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		log.Printf("error: directory does not exist: %v", directory)
		t.Fatal(err)
	}
}

func GetAutomaticDestinationsOrFatal(t *testing.T) []string {
	currentDirectory := GetCurrentDirectoryOrFatal(t)
	automaticDestinationsDirectory := filepath.Join(currentDirectory, "AutomaticDestinations")
	EnsureDirectoryExistsOrFatal(t, automaticDestinationsDirectory)

	files, err := os.ReadDir(automaticDestinationsDirectory)
	if err != nil {
		log.Printf("error: failed to read AutomaticDestinations directory: %v", err)
		t.Fatal(err)
	}

	automaticDestinations := []string{}
	for _, file := range files {
		automaticDestinations = append(automaticDestinations, filepath.Join(automaticDestinationsDirectory, file.Name()))
	}
	return automaticDestinations
}

func GetCustomDestinationsOrFatal(t *testing.T) []string {
	currentDirectory := GetCurrentDirectoryOrFatal(t)
	customDestinationsDirectory := filepath.Join(currentDirectory, "CustomDestinations")
	EnsureDirectoryExistsOrFatal(t, customDestinationsDirectory)

	files, err := os.ReadDir(customDestinationsDirectory)
	if err != nil {
		log.Printf("error: failed to read CustomDestinations directory: %v", err)
		t.Fatal(err)
	}
	customDestinations := []string{}
	for _, file := range files {
		customDestinations = append(customDestinations, filepath.Join(customDestinationsDirectory, file.Name()))
	}
	return customDestinations
}