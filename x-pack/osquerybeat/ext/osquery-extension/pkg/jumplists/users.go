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

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

// UserProfile represents a user profile on the system.
type UserProfile struct {
	Username        string `osquery:"username"`
	Domain          string `osquery:"domain"`
	Sid             string `osquery:"sid"`
	recentDirectory string
}

// profileListKey is the registry key that contains the list of user profiles. Each
// user profile is represented by a subkey with the user's SID as the name.
const profileListKey = `SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList`

// getFilesInDirectory is a helper functions that returns a list of files in a given directory.
func getFilesInDirectory(directory string, log *logger.Logger) ([]string, error) {
	fileEntries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	files := make([]string, len(fileEntries))
	for _, entry := range fileEntries {
		if entry.IsDir() {
			continue
		}
		files = append(files, filepath.Join(directory, entry.Name()))
	}
	return files, nil
}

// getJumplists returns a list of constructedjumplists for a given user profile and jumplist type.
func (u *UserProfile) getJumplists(log *logger.Logger) []*Jumplist {
	var jumplists []*Jumplist

	jumplistDirectories := map[JumplistType]string{
		JumplistTypeCustom: filepath.Join(u.recentDirectory, "CustomDestinations"),
		// Follow on PR will add support for automatic jumplists
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
		case JumplistTypeCustom:
			for _, file := range files {
				if !strings.HasSuffix(file, ".customDestinations-ms") {
					continue
				}
				jumpList, err := parseCustomJumplistFile(file, u, log)
				if err != nil {
					log.Errorf("failed to parse custom jump list file %s: %v", file, err)
					continue
				}
				jumplists = append(jumplists, jumpList)
			}
		}
	}
	return jumplists
}

// resolveSid looks up the username and domain for a given SID.
func resolveSid(sid string, log *logger.Logger) (string, string) {
	// corrupted or temporary profile keys may have a .bak suffix
	// lets make a best effort to resolve those as well
	sid = strings.TrimSuffix(sid, ".bak")

	// Convert the SID string to a SID object
	sidObj, err := windows.StringToSid(sid)
	if err != nil {
		log.Errorf("failed to convert SID %s to SID object: %v", sid, err)
		return "", ""
	}

	// Get the username and domain for the SID
	var (
		nameLen   uint32 = 256
		domainLen uint32 = 256
		use       uint32
	)
	name := make([]uint16, nameLen)
	domainName := make([]uint16, domainLen)
	err = windows.LookupAccountSid(nil, sidObj, &name[0], &nameLen, &domainName[0], &domainLen, &use)
	if err != nil {
		log.Errorf("failed to lookup account for SID %s: %v", sid, err)
		return "", ""
	}

	// Convert the username and domain to strings
	username := windows.UTF16ToString(name)
	domain := windows.UTF16ToString(domainName)

	return username, domain
}

// getUserProfiles returns a list of user profiles on the system.
func getUserProfiles(log *logger.Logger) ([]*UserProfile, error) {
	// Open the ProfileList key in HKEY_LOCAL_MACHINE
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, profileListKey, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
	if err != nil {
		return nil, err
	}
	defer k.Close()

	// Get all subkey names (User SIDs)
	sids, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return nil, err
	}

	// Iterate over the user SIDs and get the profile path for each user
	var userProfiles []*UserProfile
	for _, sid := range sids {

		// Jumplist files are stored in the user's "recent" directory, so if we can't find it, we don't want to include this user profile
		// The profile path is stored in the registry under the ProfileList\<SID>\ProfileImagePath key.
		// The "recent" directory is <profile path>\$APPDATA\Roaming\Microsoft\Windows\Recent

		// Open the registry key for the given SID
		keyPath := fmt.Sprintf(`%s\%s`, profileListKey, sid)
		sidKey, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.QUERY_VALUE)
		if err != nil {
			log.Errorf("failed to open registry key %s: %v", keyPath, err)
			continue
		}

		// Get the profile path for the given SID
		profilePath, _, err := sidKey.GetStringValue("ProfileImagePath")
		sidKey.Close() // Close immediately after reading
		if err != nil {
			log.Errorf("failed to get profile path for SID %s: %v", sid, err)
			continue
		}

		// Check if the profile path exists
		if _, err := os.Stat(profilePath); errors.Is(err, os.ErrNotExist) {
			log.Infof("profile path %s for SID %s does not exist, skipping user profile", profilePath, sid)
			continue
		}

		// Construct and resolve the "recent" directory, expanding the environment variables in the path
		recentDirectory := filepath.Join(profilePath, "AppData", "Roaming", "Microsoft", "Windows", "Recent")

		// Check if the "recent" directory exists, if it doesn't, we don't want to include this user profile
		if _, err := os.Stat(recentDirectory); errors.Is(err, os.ErrNotExist) {
			log.Infof("recent directory %s for SID %s does not exist, skipping user profile", recentDirectory, sid)
			continue
		}

		// Resolve the SID to a username and domain, we are making a best effort, so it is ok if we don't have a username and domain for a given SID.
		// but we don't want to include the SYSTEM account or any accounts in the NT AUTHORITY domain.
		username, domain := resolveSid(sid, log)
		if username == "SYSTEM" || domain == "NT AUTHORITY" {
			continue
		}

		// At this point we have a valid user profile with a valid recent directory
		userProfile := &UserProfile{
			Username:        username,
			Domain:          domain,
			Sid:             sid,
			recentDirectory: recentDirectory,
		}

		userProfiles = append(userProfiles, userProfile)
	}
	return userProfiles, nil
}
