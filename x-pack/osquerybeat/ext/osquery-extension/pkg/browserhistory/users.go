// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func discoverUsers(log func(m string, kvs ...any)) []string {
	var userDirs []string
	var searchPaths []string

	log("discoverUsers called", "os", runtime.GOOS)

	switch runtime.GOOS {
	case "windows":
		systemDrive := os.Getenv("SYSTEMDRIVE")
		if systemDrive == "" {
			systemDrive = "C:"
		}
		searchPaths = []string{
			systemDrive + "\\Users\\*",
		}
	case "darwin":
		searchPaths = []string{
			"/Users/*",
			"/var/root",
		}
	case "linux":
		searchPaths = []string{
			"/home/*",
			"/root",
		}
	default:
		return nil
	}

	for _, searchPath := range searchPaths {
		matches, err := filepath.Glob(searchPath)
		if err != nil {
			log("glob failed", "searchPath", searchPath, "error", err)
			continue
		}

		for _, match := range matches {
			if info, err := os.Stat(match); err == nil && info.IsDir() {
				username := filepath.Base(match)
				if isSystemUser(username) {
					log("skipping system user", "username", username, "path", match)
					continue
				}
				found := false
				for _, existing := range userDirs {
					if existing == match { // Compare full paths, not just usernames
						found = true
						break
					}
				}
				if !found {
					log("discovered user", "username", username, "fullPath", match)
					userDirs = append(userDirs, match) // Store full path instead of just username
				}
			}
		}
	}
	return userDirs
}

func isSystemUser(username string) bool {
	lowerUsername := strings.ToLower(username)
	switch runtime.GOOS {
	case "windows":
		switch lowerUsername {
		case "public", "default", "default user",
			"desktop.ini", "guest", "all users":
			return true
		}
	case "darwin":
		switch lowerUsername {
		case "shared", ".localized", "guest":
			return true
		}
	case "linux":
		switch lowerUsername {
		case "lost+found":
			return true
		}
	}
	return false
}
