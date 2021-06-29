// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build windows

package install

import (
	"github.com/hectane/go-acl"
	"golang.org/x/sys/windows"
	"io/fs"
	"path/filepath"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
)

// fixPermissions fixes the permissions to be correct on the installed system
func fixPermissions() error {
	// fix permissions so only SYSTEM and Administrators have access to the files in the install path
	err := recursiveSystemAdminPermissions(paths.InstallPath)
	if err != nil {
		return err
	}
	return nil
}

func recursiveSystemAdminPermissions(path string) error {
	return filepath.Walk(path, func(name string, info fs.FileInfo, err error) error {
		if err == nil {
			// first level doesn't inherit
			inherit := true
			if path == name {
				inherit = false
			}
			return systemAdministratorsOnly(name, inherit)
		}
		return err
	})
}

func systemAdministratorsOnly(path string, inherit bool) error {
	systemSID, err := windows.StringToSid("S-1-5-18")
	if err != nil {
		return err
	}
	administratorsSID, err := windows.StringToSid("S-1-5-32-544")
	if err != nil {
		return err
	}
	return acl.Apply(
		path, true, inherit,
		acl.GrantSid(0xF10F0000, systemSID),  // full control of all acl's
		acl.GrantSid(0xF10F0000, administratorsSID))
}
