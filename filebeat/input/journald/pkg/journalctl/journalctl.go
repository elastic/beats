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

package journalctl

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

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
}

// Factory returns an instance of journalctl ready to use.
// The caller is responsible for calling Kill to ensure the
// journalctl process created is correctly terminated.
//
// The returned type is an interface to allow mocking for testing
func Factory(canceller input.Canceler, logger *logp.Logger, binary string, args ...string) (Jctl, error) {
	cmd := exec.Command(binary, args...)

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
					logger.Errorf("cannot read from journalctl stdout: %s", err)
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

	logger.Infof("Journalctl command: journalctl %s", strings.Join(args, " "))

	if err := cmd.Start(); err != nil {
		return &journalctl{}, fmt.Errorf("cannot start journalctl: %w", err)
	}

	logger.Infof("journalctl started with PID %d", cmd.Process.Pid)

	// Whenever the journalctl process exits, the `Wait` call returns,
	// if there was an error it is logged and this goroutine exits.
	go func() {
		if err := cmd.Wait(); err != nil {
			jctl.logger.Errorf("journalctl exited with an error, exit code %d ", cmd.ProcessState.ExitCode())
		}
	}()

	return &jctl, nil
}

// Kill Terminates the journalctl process using a SIGKILL.
func (j *journalctl) Kill() error {
	j.logger.Debug("sending SIGKILL to journalctl")
	err := j.cmd.Process.Kill()
	return err
}

func (j *journalctl) Next(cancel input.Canceler) ([]byte, error) {
	select {
	case <-cancel.Done():
		return []byte{}, ErrCancelled
	case d, open := <-j.dataChan:
		if !open {
			return []byte{}, errors.New("no more data to read, journalctl might have exited unexpectedly")
		}
		return d, nil
	}
}
