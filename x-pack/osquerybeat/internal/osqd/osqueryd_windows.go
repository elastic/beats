// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows
// +build windows

package osqd

import (
	"fmt"
	"os/exec"
	"syscall"

	"github.com/gofrs/uuid"
)

const (
	extensionName = "osquery-extension.exe"
)

func CreateSocketPath() (string, func(), error) {
	return SocketPath(""), func() {
	}, nil
}

func SocketPath(dir string) string {
	return `\\.\pipe\elastic\osquery\` + uuid.Must(uuid.NewV4()).String()
}

func platformArgs() map[string]interface{} {
	return map[string]interface{}{
		"allow_unsafe": true,
	}
}

func setpgid() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}

// Borrowed from https://github.com/kolide/launcher/blob/master/pkg/osquery/runtime/runtime_helpers_windows.go#L25
// For clean process tree kill
func killProcessGroup(cmd *exec.Cmd) error {
	// https://github.com/golang/dep/pull/857
	exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprint(cmd.Process.Pid)).Run()
	return nil
}
