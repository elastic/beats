// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows
// +build windows

package process

import (
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

const (
	// exitCodeStillActive according to docs.microsoft.com/en-us/windows/desktop/api/processthreadsapi/nf-processthreadsapi-getexitcodeprocess
	exitCodeStillActive = 259
)

// externalProcess is a watch mechanism used in cases where OS requires
// a process to be a child for waiting for process. We need to be able
// await any process
func (a *Application) externalProcess(proc *os.Process) {
	if proc == nil {
		return
	}

	for {
		<-time.After(1 * time.Second)
		if isWindowsProcessExited(proc.Pid) {
			return
		}
	}
}

func isWindowsProcessExited(pid int) bool {
	const desiredAccess = syscall.STANDARD_RIGHTS_READ | syscall.PROCESS_QUERY_INFORMATION | syscall.SYNCHRONIZE
	h, err := windows.OpenProcess(desiredAccess, false, uint32(pid)) //nolint:gosec // G115 Conversion from int to uint32 is safe here.
	if err != nil {
		// failed to open handle, report exited
		return true
	}
	defer windows.CloseHandle(h) //nolint:errcheck // No way to handle errors returned here so safe to ignore.

	// get exit code, this returns immediately in case it is still running
	// it returns exitCodeStillActive
	var ec uint32
	if err := windows.GetExitCodeProcess(h, &ec); err != nil {
		// failed to contact, report exited
		return true
	}

	return ec != exitCodeStillActive
}
