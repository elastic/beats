// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build windows

package reexec

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// exec performs execution on Windows.
//
// Windows does not support the ability to execute over the same PID and memory. Depending on the execution context
// different scenarios need to occur.
//
// * Services.msc - A new child process is spawned that waits for the service to stop, then restarts it and the
//   current process just exits.
//
// * Sub-process - As a sub-process a new child is spawned and the current process just exits.
func exec(exec string) error {
	svc, status, err := getService()
	if err == nil {
		// running as a service; spawn re-exec windows sub-process
		args := []string{filepath.Base(exec), "reexec_windows", svc.Name, strconv.Itoa(status.ProcessId)}
		cmd := exec.Cmd{
			Path:   exec,
			Args:   args,
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}
		if err := cmd.Start(); err != nil {
			return err
		}
	} else {
		// running as a sub-process of another process; just execute as a child
		args := []string{filepath.Base(exec)}
		args = append(args, os.Args[1:]...)
		cmd := exec.Cmd{
			Path:   exec,
			Args:   args,
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}
		if err := cmd.Start(); err != nil {
			return err
		}
	}
	os.Exit(0)
	return nil
}

func getService() (*mgr.Service, svc.Status, error) {
	pid := os.Getpid()
	manager, err := mgr.Connect()
	if err != nil {
		return nil, svc.Status{}, err
	}
	names, err := manager.ListServices()
	if err != nil {
		return nil, svc.Status{}, err
	}
	for _, name := range names {
		service, err := manager.OpenService(name)
		if err != nil {
			continue
		}
		status, err := service.Query()
		if err != nil {
			continue
		}
		if status.ProcessId == pid {
			// pid match; found ourself
			return service, status, nil
		}
	}
	return nil, svc.Status{}, fmt.Errorf("failed to find service")
}
