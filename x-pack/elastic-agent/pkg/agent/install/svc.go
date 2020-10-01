// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package install

import (
	"path/filepath"

	"github.com/kardianos/service"
)

const (
	// ServiceDisplayName is the service display name for the service.
	ServiceDisplayName = "Elastic Agent"

	// ServiceDescription is the description for the service.
	ServiceDescription = "Elastic Agent is a unified agent to observe, monitor and protect your system."
)

// ExecutablePath returns the path for the installed Agents executable.
func ExecutablePath() string {
	exec := filepath.Join(InstallPath, BinaryName)
	if ShellWrapperPath != "" {
		exec = ShellWrapperPath
	}
	return exec
}

func newService() (service.Service, error) {
	return service.New(nil, &service.Config{
		Name:             ServiceName,
		DisplayName:      ServiceDisplayName,
		Description:      ServiceDescription,
		Executable:       ExecutablePath(),
		WorkingDirectory: InstallPath,
	})
}
