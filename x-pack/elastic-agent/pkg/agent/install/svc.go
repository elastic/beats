package install

import (
	"path/filepath"

	"github.com/kardianos/service"
)

const (
	// ServiceDisplayName is the service display name for the service.
	ServiceDisplayName = "Elastic Agent"

	ServiceDescription = `
Elastic Agent is a unified agent to observe, monitor and protect your system.
`
)

func newService() (service.Service, error) {
	exec := filepath.Join(InstallPath, BinaryName)
	if ShellWrapperPath != "" {
		exec = ShellWrapperPath
	}
	return service.New(nil, &service.Config{
		Name:             ServiceName,
		DisplayName:      ServiceDisplayName,
		Description:      ServiceDescription,
		Executable:       exec,
		WorkingDirectory: InstallPath,
	})
}
