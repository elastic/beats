// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build windows

package install

const (
	// BinaryName is the name of the installed binary.
	BinaryName = "elastic-agent.exe"

	// InstallPath is the installation path using for install command.
	InstallPath = `C:\Program Files\Elastic\Agent`

	// SocketPath is the socket path used when installed.
	//
	// `\\.\pipe\elastic-agent-%x`, sha256.Sum256([]byte(`C:\Program Files\Elastic\Agent\elastic-agent.exe`))
	SocketPath = `\\.\pipe\elastic-agent-56c56575c1f574fe48db8f56ac6db1cbcd78996a355bd2b44c71ebedd9c9a15b`

	// ServiceName is the service name when installed.
	ServiceName = "Elastic Agent"

	// ShellWrapperPath is the path to the installed shell wrapper.
	ShellWrapperPath = "" // no wrapper on Windows

	// ShellWrapper is the wrapper that is installed.
	ShellWrapper = "" // no wrapper on Windows
)
