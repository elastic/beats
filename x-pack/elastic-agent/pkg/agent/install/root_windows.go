// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build windows

package install

import (
	"golang.org/x/sys/windows"
)

const (
	// PermissionUser is the permission level the user needs to be.
	PermissionUser = "Administrator"
)

// HasRoot returns true if the user has Administrator/SYSTEM permissions.
func HasRoot() bool {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	token := windows.Token(0)
	_, err = token.IsMember(sid)
	if err != nil {
		return false
	}
	return token.IsElevated()
}
