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
