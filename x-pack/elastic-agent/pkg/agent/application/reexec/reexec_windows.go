// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build windows

package reexec

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
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
func reexec(log *logger.Logger, executable string, argOverrides ...string) error {
	svc, status, err := getService()
	if err == nil {
		// running as a service; spawn re-exec windows sub-process
		log.Infof("Running as Windows service %s; triggering service restart", svc.Name)
		args := []string{filepath.Base(executable), "reexec_windows", svc.Name, strconv.Itoa(int(status.ProcessId))}
		args = append(args, argOverrides...)
		cmd := exec.Cmd{
			Path:   executable,
			Args:   args,
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}
		if err := cmd.Start(); err != nil {
			return err
		}
	} else {
		log.Debugf("Discovering Windows service result: %s", err)

		// running as a sub-process of another process; just execute as a child
		log.Infof("Running as Windows process; spawning new child process")
		args := []string{filepath.Base(executable)}
		args = append(args, os.Args[1:]...)
		args = append(args, argOverrides...)
		cmd := exec.Cmd{
			Path:   executable,
			Args:   args,
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}
		if err := cmd.Start(); err != nil {
			return err
		}
	}
	// force log sync before exit
	_ = log.Sync()
	return nil
}

func getService() (*mgr.Service, svc.Status, error) {
	pid := uint32(os.Getpid())
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
