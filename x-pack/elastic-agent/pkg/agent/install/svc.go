// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package install

import (
	"path/filepath"

	"github.com/kardianos/service"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
)

const (
	// ServiceDisplayName is the service display name for the service.
	ServiceDisplayName = "Elastic Agent"

	// ServiceDescription is the description for the service.
	ServiceDescription = "Elastic Agent is a unified agent to observe, monitor and protect your system."
)

// ExecutablePath returns the path for the installed Agents executable.
func ExecutablePath() string {
	exec := filepath.Join(paths.InstallPath, paths.BinaryName)
	if paths.ShellWrapperPath != "" {
		exec = paths.ShellWrapperPath
	}
	return exec
}

func newService() (service.Service, error) {
	return service.New(nil, &service.Config{
		Name:             paths.ServiceName,
		DisplayName:      ServiceDisplayName,
		Description:      ServiceDescription,
		Executable:       ExecutablePath(),
		WorkingDirectory: paths.InstallPath,
		Option: map[string]interface{}{
			// Linux (systemd) always restart on failure
			"Restart": "always",

			// Windows setup restart on failure
			"OnFailure":              "restart",
			"OnFailureDelayDuration": "1s",
			"OnFailureResetPeriod":   10,
		},
	})
}
