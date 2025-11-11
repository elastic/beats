// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package testdata

import (
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func GetPredictableTime(seed int64) time.Time {
	// Create a new source of randomness using the seed
	r := rand.New(rand.NewSource(seed))

	// Generate random but predictable components for the time
	year := 2000 + r.Intn(25) // Random year between 2000-2024
	month := 1 + r.Intn(12)   // Random month 1-12
	day := 1 + r.Intn(28)     // Random day 1-28 (avoiding month length issues)
	hour := r.Intn(24)        // Random hour 0-23
	minute := r.Intn(60)      // Random minute 0-59
	second := r.Intn(60)      // Random second 0-59

	return time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC)
}

func MustGetTestDataDirectory(t *testing.T) string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get current file path")
	}
	return filepath.Dir(currentFile)
}

func MustEnsureDirectoryExists(t *testing.T, directory string) {
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		log.Printf("error: directory does not exist: %v", directory)
		t.Fatal(err)
	}
}

func MustGetFilesInDirectory(t *testing.T, directory string) []string {
	MustEnsureDirectoryExists(t, directory)

	absPath, err := filepath.Abs(directory)
	if err != nil {
		log.Printf("error: failed to get absolute path: %v", err)
		t.Fatal(err)
	}
	files, err := os.ReadDir(absPath)
	if err != nil {
		log.Printf("error: failed to read directory: %v", err)
		t.Fatal(err)
	}
	filesNames := []string{}
	for _, file := range files {
		filesNames = append(filesNames, filepath.Join(absPath, file.Name()))
	}
	return filesNames
}

func MustGetCustomDestinationDirectory(t *testing.T) string {
	testDataDirectory := MustGetTestDataDirectory(t)
	return filepath.Join(testDataDirectory, "custom_destinations")
}

func MustGetAutomaticDestinationDirectory(t *testing.T) string {
	testDataDirectory := MustGetTestDataDirectory(t)
	return filepath.Join(testDataDirectory, "automatic_destinations")
}

func MustGetAutomaticDestinations(t *testing.T) []string {
	automaticDestinationsDirectory := MustGetAutomaticDestinationDirectory(t)
	return MustGetFilesInDirectory(t, automaticDestinationsDirectory)
}

func MustGetCustomDestinations(t *testing.T) []string {
	customDestinationsDirectory := MustGetCustomDestinationDirectory(t)
	return MustGetFilesInDirectory(t, customDestinationsDirectory)
}
