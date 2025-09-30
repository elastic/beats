// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows
// +build windows

package server

import (
	"fmt"
	"net"
	"os/user"
	"runtime/debug"

	"github.com/pkg/errors"
	"golang.org/x/sys/windows"

	"github.com/elastic/beats/v7/libbeat/api/npipe"
	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

// createListener creates a named pipe listener on Windows
func createListener(_ *logger.Logger) (net.Listener, error) {
	sd, err := securityDescriptor()
	if err != nil {
		return nil, err
	}
	return npipe.NewListener(control.Address(), sd)
}

func cleanupListener(_ *logger.Logger) {
	// nothing to do on windows
}

func securityDescriptor() (string, error) {
	var u *user.User
	var err error
	logger := logp.L().Named("EA-npipe")

	bi, ok := debug.ReadBuildInfo()
	logger.Infof("==================== Go version: %s. OK? %t", bi.GoVersion, ok)

	u, err = user.Current()
	if err != nil {
		return "", errors.Wrap(err, "failed to retrieve the current user")
	}
	logger.Infof("==================== user.Current: Username: %q, Name: %q, Uid: %q Gid: %q, HomeDir: %q",
		u.Username,
		u.Name,
		u.Uid,
		u.Gid,
		u.HomeDir)

	// Named pipe security and access rights.
	// We create the pipe and the specific users should only be able to write to it.
	// See docs: https://docs.microsoft.com/en-us/windows/win32/ipc/named-pipe-security-and-access-rights
	// String definition: https://docs.microsoft.com/en-us/windows/win32/secauthz/ace-strings
	// Give generic read/write access to the specified user.
	descriptor := "D:P(A;;GA;;;" + u.Uid + ")"
	isAdmin, err := hasRoot()
	if err != nil {
		// do not fail, agent would end up in a loop, continue with limited permissions
		logp.Warn("failed to detect Administrator: %v", err)
	}
	logger.Infof("==================== isAdmin: %t", isAdmin)
	if isAdmin {
		// running as SYSTEM, include Administrators group so Administrators can talk over
		// the named pipe to the running Elastic Agent system process
		// https://support.microsoft.com/en-us/help/243330/well-known-security-identifiers-in-windows-operating-systems
		descriptor += "(A;;GA;;;S-1-5-32-544)" // Administrators group
	}

	old, err := oldSD()
	logger.Infof("============================== descriptor: %q", descriptor)
	logger.Infof("============================== Old Descriptor: %q. Error: %s", old, err)

	return descriptor, nil
}

func oldSD() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", errors.Wrap(err, "failed to get current user")
	}
	// Named pipe security and access rights.
	// We create the pipe and the specific users should only be able to write to it.
	// See docs: https://docs.microsoft.com/en-us/windows/win32/ipc/named-pipe-security-and-access-rights
	// String definition: https://docs.microsoft.com/en-us/windows/win32/secauthz/ace-strings
	// Give generic read/write access to the specified user.
	descriptor := "D:P(A;;GA;;;" + u.Uid + ")"
	if u.Username == "NT AUTHORITY\\SYSTEM" {
		// running as SYSTEM, include Administrators group so Administrators can talk over
		// the named pipe to the running Elastic Agent system process
		// https://support.microsoft.com/en-us/help/243330/well-known-security-identifiers-in-windows-operating-systems
		descriptor += "(A;;GA;;;S-1-5-32-544)" // Administrators group
	}
	return descriptor, nil
}

// administratorSID is the SID for the Administrator user.
const administratorSID = "S-1-5-32-544"

// hasRoot returns true if the user has Administrator/SYSTEM permissions.
func hasRoot() (bool, error) {
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
		return false, fmt.Errorf("allocate sid error: %w", err)
	}
	defer func() {
		_ = windows.FreeSid(sid)
	}()

	token := windows.Token(0)

	member, err := token.IsMember(sid)
	if err != nil {
		return false, fmt.Errorf("token membership error: %w", err)
	}

	return member, nil
}
