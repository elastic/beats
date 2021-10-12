// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows
// +build windows

package paths

import "strings"

const (
	// BinaryName is the name of the installed binary.
	BinaryName = "elastic-agent.exe"

	// InstallPath is the installation path using for install command.
	InstallPath = `C:\Program Files\Elastic\Agent`

	// SocketPath is the socket path used when installed.
	SocketPath = `\\.\pipe\elastic-agent-system`

	// ServiceName is the service name when installed.
	ServiceName = "Elastic Agent"

	// ShellWrapperPath is the path to the installed shell wrapper.
	ShellWrapperPath = "" // no wrapper on Windows

	// ShellWrapper is the wrapper that is installed.
	ShellWrapper = "" // no wrapper on Windows
)

// ArePathsEqual determines whether paths are equal taking case sensitivity of os into account.
func ArePathsEqual(expected, actual string) bool {
	return strings.EqualFold(expected, actual)
}
