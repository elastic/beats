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

//go:build linux

package journalctl

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/elastic-agent-libs/logp"
)

type journalctl struct {
	cmd      *exec.Cmd
	dataChan chan []byte
	stdout   io.ReadCloser
	stderr   io.ReadCloser

	logger   *logp.Logger
	canceler input.Canceler
	waitDone sync.WaitGroup
}

// NewFactory returns a function that instantiates [journalctl].
// The caller is responsible for calling Kill to ensure the
// journalctl process created is correctly terminated.
//
// If chroot is non-empty, then journalctlPath must be non-empty and be
// the absolute path to the journalctl binary inside the chroot.
//
// The returned type is an interface to allow mocking for testing
func NewFactory(chroot, journalctlPath string) JctlFactory {
	return func(canceller input.Canceler, logger *logp.Logger, args ...string) (Jctl, error) {
		cmd := exec.Command(journalctlPath, args...)

		if chroot != "" {
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Chroot: chroot,
			}
		}

		jctl := journalctl{
			canceler: canceller,
			cmd:      cmd,
			dataChan: make(chan []byte),
			logger:   logger,
		}

		var err error
		jctl.stdout, err = cmd.StdoutPipe()
		if err != nil {
			return &journalctl{}, fmt.Errorf("cannot get stdout pipe: %w", err)
		}
		jctl.stderr, err = cmd.StderrPipe()
		if err != nil {
			return &journalctl{}, fmt.Errorf("cannot get stderr pipe: %w", err)
		}

		// This gorroutune reads the stderr from the journalctl process, if the
		// process exits for any reason, then its stderr is closed, this goroutine
		// gets an EOF error and exits
		go func() {
			defer jctl.logger.Debug("stderr reader goroutine done")
			reader := bufio.NewReader(jctl.stderr)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if !errors.Is(err, io.EOF) {
						logger.Errorf("cannot read from journalctl stderr: %s", err)
					}
					return
				}

				logger.Errorf("Journalctl wrote to stderr: %s", line)
			}
		}()

		// This goroutine reads the stdout from the journalctl process and makes
		// the data available via the `Next()` method.
		// If the journalctl process exits for any reason, then its stdout is closed
		// this goroutine gets an EOF error and exits.
		go func() {
			defer jctl.logger.Debug("stdout reader goroutine done")
			defer close(jctl.dataChan)
			reader := bufio.NewReader(jctl.stdout)
			for {
				data, err := reader.ReadBytes('\n')
				if err != nil {
					if !errors.Is(err, io.EOF) {
						var logError = false
						var pathError *fs.PathError
						if errors.As(err, &pathError) {
							// Because we're reading from the stdout from a process that will
							// eventually exit, it can happen that when reading we get the
							// fs.PathError below instead of an io.EOF. This is expected,
							// it only means the process has exited, its stdout has been
							// closed and there is nothing else for us to read.
							// This is expected and does not cause any data loss.
							// So we log at level debug to have it in our logs if ever needed
							// while avoiding adding error level logs on user's deployments
							// for situations that are well handled.
							if pathError.Op == "read" &&
								pathError.Path == "|0" &&
								pathError.Err.Error() == "file already closed" {
								logger.Debugf("cannot read from journalctl stdout: '%s'", err)
							} else {
								logError = true
							}
						} else {
							logError = true
						}
						if logError {
							logger.Errorf("cannot read from journalctl stdout: '%s'", err)
						}
					}
					return
				}

				select {
				case <-jctl.canceler.Done():
					return
				case jctl.dataChan <- data:
				}
			}
		}()

		processCmdLine := strings.Join(append([]string{journalctlPath}, args...), " ")

		logger.Infow(
			"Journalctl command. Paths relative to chroot (if set)",
			"process.command_line", processCmdLine,
			"process.chroot", chroot,
		)

		if err := cmd.Start(); err != nil {
			return &journalctl{}, fmt.Errorf("cannot start journalctl: %w. Chroot: %s", err, chroot)
		}

		logger.Infow(
			"journalctl started",
			"process.pid", cmd.Process.Pid,
		)
		jctl.logger = jctl.logger.With(
			"process.pid", cmd.Process.Pid,
		)

		// Whenever the journalctl process exits, the `Wait` call returns,
		// if there was an error it is logged and this goroutine exits.
		jctl.waitDone.Add(1)
		go func() {
			defer jctl.waitDone.Done()
			if err := cmd.Wait(); err != nil {
				jctl.logger.Errorf("journalctl exited with an error, exit code %d ", cmd.ProcessState.ExitCode())
			}
			jctl.logger.Debugf("journalctl exit code: %d", cmd.ProcessState.ExitCode())
		}()

		return &jctl, nil
	}
}

// Kill Terminates the journalctl process using a SIGKILL.
func (j *journalctl) Kill() error {
	j.logger.Debug("sending SIGKILL to journalctl")
	return j.cmd.Process.Kill()
}

// Next returns the next journal entry (as JSON). If `finished` is true, then
// journalctl finished returning all data and exited successfully, if journalctl
// exited unexpectedly, then `err` is non-nil, `finished` is false and an empty
// byte array is returned.
func (j *journalctl) Next(cancel input.Canceler) ([]byte, error) {
	select {
	case <-cancel.Done():
		return []byte{}, ErrCancelled
	case d, open := <-j.dataChan:
		if !open {
			// Wait for the process to exit, so we can read the exit code.
			j.waitDone.Wait()
			return []byte{},
				fmt.Errorf(
					"no more data to read, journalctl exited unexpectedly, exit code: %d",
					j.cmd.ProcessState.ExitCode())
		}

		return d, nil
	}
}
