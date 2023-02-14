// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build windows
// +build windows

package instance

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"github.com/elastic/elastic-agent-libs/logp"
)

// doReexec performs execution on Windows.
//
// Windows does not support the ability to execute over the same PID and memory. Depending on the execution context
// different scenarios need to occur.
//
//   - Services.msc - A new child process is spawned that waits for the service to stop, then restarts it and the
//     current process just exits.
//
// * Sub-process - As a sub-process a new child is spawned and the current process just exits.
func (b *Beat) doReexec() error {
	logger := logp.L().Named(b.Info.Beat)
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get working directory: %w", err)
	}

	executable := filepath.Join(pwd, os.Args[0])

	svc, status, err := getService()
	if err == nil {
		// running as a service; spawn re-exec windows sub-process
		logger.Infof("Running as Windows service %s; triggering service restart", svc.Name)
		args := []string{filepath.Base(executable), "reexec_windows", svc.Name, strconv.Itoa(int(status.ProcessId))}
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
		logger.Debugf("Discovering Windows service result: %s", err)

		// running as a sub-process of another process; just execute as a child
		logger.Infof("Running as Windows process; spawning new child process")
		args := []string{filepath.Base(executable)}
		args = append(args, os.Args[1:]...)
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
	_ = logger.Sync()
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
