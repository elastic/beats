// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows

package osqd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

const (
	extensionName = "osquery-extension.ext"
)

func CreateSocketPath() (string, func(), error) {
	// Try to create socket in /var/run first
	// This would result in something the directory something like: /var/run/027202467
	tpath, err := os.MkdirTemp("/var/run", "")
	if err != nil {
		var perr *os.PathError
		if errors.As(err, &perr) {
			if errors.Is(perr.Err, syscall.EACCES) {
				tpath, err = os.MkdirTemp("", "")
				if err != nil {
					return "", nil, err
				}
			}
		}
	}

	return SocketPath(tpath), func() {
		os.RemoveAll(tpath)
	}, nil
}

func SocketPath(dir string) string {
	return filepath.Join(dir, "osquery.sock")
}

func platformArgs() map[string]interface{} {
	return nil
}

func setpgid() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

// Borrowed from https://github.com/kolide/launcher/blob/master/pkg/osquery/runtime/runtime_helpers.go#L20
// For clean process tree kill
func killProcessGroup(cmd *exec.Cmd) error {
	err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	if err != nil {
		return fmt.Errorf("kill process group %d, %w", cmd.Process.Pid, err)
	}
	return nil
}
