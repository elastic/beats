// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !windows

package osqueryd

import (
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
)

func SocketPath(dir string) string {
	return filepath.Join(dir, "osquery.sock")
}

func platformArgs() []string {
	return nil
}

func setpgid() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

// Borrowed from https://github.com/kolide/launcher/blob/master/pkg/osquery/runtime/runtime_helpers.go#L20
// For clean process tree kill
func killProcessGroup(cmd *exec.Cmd) error {
	err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	return errors.Wrapf(err, "kill process group %d", cmd.Process.Pid)
}
