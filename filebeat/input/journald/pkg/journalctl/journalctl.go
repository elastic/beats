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
	"os"
	"os/exec"
	"strings"
	"sync"

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

	// Stop chan and StopOnce are used to ensure the stdout reader goroutine
	// can stop even if nobody is reading from the dataChan.
	stopCh   chan struct{}
	stopOnce sync.Once
}

// Factory returns an instance of journalctl ready to use.
// The caller is responsible for calling Kill to ensure the
// journalctl process created is correctly terminated.
//
// The returned type is an interface to allow mocking for testing
func Factory(canceller input.Canceler, logger *logp.Logger, binary string, args ...string) (Jctl, error) {
	//nolint:noctx // we use the canceller to correctly stop the process
	cmd := exec.Command(binary, args...)

	jctl := journalctl{
		canceler: canceller,
		cmd:      cmd,
		dataChan: make(chan []byte),
		logger:   logger,
	}

<<<<<<< HEAD
	var err error
	jctl.stdout, err = cmd.StdoutPipe()
	if err != nil {
		return &journalctl{}, fmt.Errorf("cannot get stdout pipe: %w", err)
	}
	jctl.stderr, err = cmd.StderrPipe()
	if err != nil {
		return &journalctl{}, fmt.Errorf("cannot get stderr pipe: %w", err)
	}

	logger.Infof("Journalctl command: %s %s", binary, strings.Join(args, " "))
=======
		jctl := journalctl{
			canceler: canceller,
			cmd:      cmd,
			dataChan: make(chan []byte),
			logger:   logger,
			stopCh:   make(chan struct{}),
		}

		var err error
		jctl.stdout, err = cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("cannot get stdout pipe: %w", err)
		}
		jctl.stderr, err = cmd.StderrPipe()
		if err != nil {
			return nil, fmt.Errorf("cannot get stderr pipe: %w", err)
		}
>>>>>>> e0d13d181 (Fix journalctl process lifecycle and cleanup bugs (#49528))

	// Start the process before trying to read from the pipes.
	// See: https://pkg.go.dev/os/exec#example-Cmd.StdoutPipe
	if err := cmd.Start(); err != nil {
		return &journalctl{}, fmt.Errorf("cannot start journalctl: %w", err)
	}

	logger.Infof("journalctl started with PID %d", cmd.Process.Pid)

<<<<<<< HEAD
	// readersWG tracks when stdout/stderr reader goroutines are done.
	// cmd.Wait must not be called until reads from StdoutPipe and StderrPipe
	// have completed (per the os/exec docs), so waitDone waits on readersWG.
	var readersWG sync.WaitGroup
=======
		// Start the process before trying to read from the pipes
		// See: https://pkg.go.dev/os/exec#example-Cmd.StdoutPipe
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("cannot start journalctl: %w. Chroot: %s", err, chroot)
		}
>>>>>>> e0d13d181 (Fix journalctl process lifecycle and cleanup bugs (#49528))

	// This gorroutune reads the stderr from the journalctl process, if the
	// process exits for any reason, then its stderr is closed, this goroutine
	// gets an EOF error and exits
	readersWG.Add(1)
	go func() {
		defer readersWG.Done()
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

<<<<<<< HEAD
			logger.Errorf("Journalctl wrote to stderr: %s", line)
		}
	}()

	// This goroutine reads the stdout from the journalctl process and makes
	// the data available via the `Next()` method.
	// If the journalctl process exits for any reason, then its stdout is closed
	// this goroutine gets an EOF error and exits.
	readersWG.Add(1)
	go func() {
		defer readersWG.Done()
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
=======
		// This goroutine reads the stdout from the journalctl process and makes
		// the data available via the `Next()` method.
		// If the journalctl process exits for any reason, then its stdout is closed
		// this goroutine gets an EOF error and exits.
		readersWG.Go(func() {
			defer jctl.logger.Debug("stdout reader goroutine done")
			defer close(jctl.dataChan)
			reader := bufio.NewReader(jctl.stdout)
			for {
				data, err := reader.ReadBytes('\n')
				if err != nil {
					if !errors.Is(err, io.EOF) {
						logger.Errorf("cannot read from journalctl stdout: '%s'", err)
					}
					return
				}

				select {
				case <-jctl.canceler.Done():
					return
				case <-jctl.stopCh:
					return
				case jctl.dataChan <- data:
>>>>>>> e0d13d181 (Fix journalctl process lifecycle and cleanup bugs (#49528))
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

	// Whenever the journalctl process exits, the `Wait` call returns,
	// if there was an error it is logged and this goroutine exits.
	// We must wait for the reader goroutines to finish before calling
	// cmd.Wait, because Wait closes the pipes obtained via StdoutPipe
	// and StderrPipe. Calling Wait prematurely causes readers to see
	// "file already closed" instead of the process output.
	jctl.waitDone.Add(1)
	go func() {
		defer jctl.waitDone.Done()
		readersWG.Wait()
		if err := cmd.Wait(); err != nil {
			jctl.logger.Errorf("journalctl exited with an error, exit code %d ", cmd.ProcessState.ExitCode())
		}
		jctl.logger.Debugf("journalctl exit code: %d", cmd.ProcessState.ExitCode())
	}()

	return &jctl, nil
}

// Kill terminates the journalctl process by sending SIGKILL, then it
// blocks until all background goroutines (stdout/stderr readers and
// the process-wait goroutine) have exited.
func (j *journalctl) Kill() error {
	j.logger.Debug("sending SIGKILL to journalctl")

	// Signal the stdout reader goroutine to exit, this ensures
	// j.waitDone.Wait() won't block if the stdout reader goroutine
	// is trying to send data and nobody is reading from its channel.
	j.stopOnce.Do(func() {
		close(j.stopCh)
	})

	err := j.cmd.Process.Kill()
	j.waitDone.Wait()
	if errors.Is(err, os.ErrProcessDone) {
		return nil
	}

	return err
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
