// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build windows

package app

import (
	"os"
	"syscall"
	"time"
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
		select {
		case <-time.After(1 * time.Second):
			if isWindowsProcessExited(proc.Pid) {
				return
			}
		}
	}
}

func isWindowsProcessExited(pid int) bool {
	const desiredAccess = syscall.STANDARD_RIGHTS_READ | syscall.PROCESS_QUERY_INFORMATION | syscall.SYNCHRONIZE
	h, err := syscall.OpenProcess(desiredAccess, false, uint32(pid))
	if err != nil {
		// failed to open handle, report exited
		return true
	}

	// get exit code, this returns immediately in case it is still running
	// it returns exitCodeStillActive
	var ec uint32
	if err := syscall.GetExitCodeProcess(h, &ec); err != nil {
		// failed to contact, report exited
		return true
	}

	return ec != exitCodeStillActive
}
