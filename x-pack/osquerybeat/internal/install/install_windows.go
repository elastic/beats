// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows
// +build windows

package install

import (
	"os/exec"
	"syscall"
)

func setCommandArg(cmd *exec.Cmd, arg string) {
	// Winders hack to pass args to msiexec without escaping
	// Set directly to avoid args escaping
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine:       " " + arg,
		HideWindow:    false,
		CreationFlags: 0,
	}
}
