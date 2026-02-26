// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

// generate the application id map
//go:generate go run ./generate

package jumplists

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

// UserProfile represents a user profile on the system.
type UserProfile struct {
	Username  string `osquery:"username"`
	Sid       string `osquery:"sid"`
	recentDir string
}

// getFilesInDirectory is a helper functions that returns a list of files in a given directory.
func getFilesInDirectory(directory string, log *logger.Logger) ([]string, error) {
	fileEntries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	files := []string{}
	for _, entry := range fileEntries {
		if entry.IsDir() {
			continue
		}
		files = append(files, filepath.Join(directory, entry.Name()))
	}
	return files, nil
}

// getJumplists returns a list of constructed jumplists for a given user profile.
func (u *UserProfile) getJumplists(log *logger.Logger) []*jumplist {
	var jumplists []*jumplist

	jumplistDirectories := map[jumplistType]string{
		jumplistTypeCustom:    filepath.Join(u.recentDir, "CustomDestinations"),
		jumplistTypeAutomatic: filepath.Join(u.recentDir, "AutomaticDestinations"),
	}

	// Collect and parse all jumplist files for each jumplist type
	for jumplistType, directory := range jumplistDirectories {
		files, err := getFilesInDirectory(directory, log)
		if err != nil {
			log.Errorf("failed to get files in directory %s: %v", directory, err)
			continue
		}

		// Parse the jumplist files for the given jumplist type
		switch jumplistType {
		case jumplistTypeCustom:
			for _, file := range files {
				if !strings.HasSuffix(file, ".customDestinations-ms") {
					continue
				}
				jumpList, err := parseCustomJumplistFile(file, u, log)
				if err != nil {
					log.Errorf("%s", err)
					continue
				}
				jumplists = append(jumplists, jumpList)
			}
		case jumplistTypeAutomatic:
			for _, file := range files {
				jumpList, err := parseAutomaticJumpListFile(file, u, log)
				if err != nil {
					log.Errorf("%s", err)
					continue
				}
				jumplists = append(jumplists, jumpList)
			}
		}
	}
	return jumplists
}

// getUserProfiles returns a list of user profiles on the system.
func getUserProfiles(log *logger.Logger, client ClientInterface) ([]*UserProfile, error) {
	var userProfiles []*UserProfile
	response, err := client.Query("SELECT * from users WHERE type != 'special' AND directory != '';")
	if err != nil {
		return nil, fmt.Errorf("failed to query users table: %w", err)
	}

	for _, userRow := range response.Response {
		profileDir := userRow["directory"]
		if profileDir == "" {
			continue
		}

		// Construct and resolve the "recent" directory, expanding the environment variables in the path
		recentDir := filepath.Join(profileDir, "AppData", "Roaming", "Microsoft", "Windows", "Recent")
		// Check if the "recent" directory exists, if it doesn't, we don't want to include this user profile
		if _, err := os.Stat(recentDir); errors.Is(err, os.ErrNotExist) {
			log.Infof("recent directory %s does not exist, skipping user profile", recentDir)
			continue
		}

		userProfile := &UserProfile{
			Username:  userRow["username"],
			Sid:       userRow["uuid"],
			recentDir: recentDir,
		}

		userProfiles = append(userProfiles, userProfile)
	}
	return userProfiles, nil
}
