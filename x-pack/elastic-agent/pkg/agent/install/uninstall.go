// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package install

import (
	"fmt"
	"os"
	"runtime"

	"github.com/kardianos/service"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
)

// Uninstall uninstalls persistently Elastic Agent on the system.
func Uninstall() error {
	// uninstall the current service
	svc, err := newService()
	if err != nil {
		return err
	}
	status, _ := svc.Status()
	if status == service.StatusRunning {
		err := svc.Stop()
		if err != nil {
			return errors.New(
				err,
				fmt.Sprintf("failed to stop service (%s)", ServiceName),
				errors.M("service", ServiceName))
		}
		status = service.StatusStopped
	}
	_ = svc.Uninstall()

	// remove, if present on platform
	if ShellWrapperPath != "" {
		err = os.Remove(ShellWrapperPath)
		if !os.IsNotExist(err) && err != nil {
			return errors.New(
				err,
				fmt.Sprintf("failed to remove shell wrapper (%s)", ShellWrapperPath),
				errors.M("destination", ShellWrapperPath))
		}
	}

	// remove existing directory
	err = os.RemoveAll(InstallPath)
	if err != nil {
		if runtime.GOOS == "windows" {
			// possible to fail on Windows, because elastic-agent.exe is running from
			// this directory.
			return nil
		}
		return errors.New(
			err,
			fmt.Sprintf("failed to remove installation directory (%s)", InstallPath),
			errors.M("directory", InstallPath))
	}

	return nil
}
