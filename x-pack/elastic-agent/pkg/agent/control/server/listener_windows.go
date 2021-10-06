// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build windows

package server

import (
	"net"
	"os/user"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/api/npipe"

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
