// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows
// +build windows

package install

import (
	"github.com/pkg/errors"
	"golang.org/x/sys/windows"
)

const (
	// PermissionUser is the permission level the user needs to be.
	PermissionUser = "Administrator"
)

// HasRoot returns true if the user has Administrator/SYSTEM permissions.
func HasRoot() (bool, error) {
	var sid *windows.SID
	// See https://docs.microsoft.com/en-us/windows/desktop/api/securitybaseapi/nf-securitybaseapi-checktokenmembership for more on the api
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {
		return false, errors.Errorf("sid error: %s", err)
	}

	token := windows.Token(0)

	member, err := token.IsMember(sid)
	if err != nil {
		return false, errors.Errorf("token membership error: %s", err)
	}

	return member, nil
}
