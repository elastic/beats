// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows
// +build windows

package operations

import (
	"golang.org/x/sys/windows"
)

const (
	ML_SYSTEM_RID = 0x4000
)

// RunningUnderSupervisor returns true when executing Agent is running under
// the supervisor processes of the OS.
func RunningUnderSupervisor() bool {
	serviceSid, err := allocSid(ML_SYSTEM_RID)
	if err != nil {
		return false
	}
	defer windows.FreeSid(serviceSid)

	t, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return false
	}
	defer t.Close()

	gs, err := t.GetTokenGroups()
	if err != nil {
		return false
	}

	for _, g := range gs.AllGroups() {
		if windows.EqualSid(g.Sid, serviceSid) {
			return true
		}
	}
	return false
}

func allocSid(subAuth0 uint32) (*windows.SID, error) {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(&windows.SECURITY_MANDATORY_LABEL_AUTHORITY,
		1, subAuth0, 0, 0, 0, 0, 0, 0, 0, &sid)
	if err != nil {
		return nil, err
	}
	return sid, nil
}
