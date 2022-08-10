// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows
// +build !windows

package osqd

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
)

const (
	extensionName = "osquery-extension.ext"
)

func CreateSocketPath() (string, func(), error) {
	// Try to create socket in /var/run first
	// This would result in something the directory something like: /var/run/027202467
	tpath, err := ioutil.TempDir("/var/run", "")
	if err != nil {
		if perr, ok := err.(*os.PathError); ok {
			if perr.Err == syscall.EACCES {
				tpath, err = ioutil.TempDir("", "")
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
	return errors.Wrapf(err, "kill process group %d", cmd.Process.Pid)
}
