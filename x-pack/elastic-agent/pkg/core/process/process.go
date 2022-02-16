// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package process

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/proc"
)

var (
	// ErrProcessStartFailedTimeout is a failure of start due to timeout
	ErrProcessStartFailedTimeout = errors.New("process failed to start due to timeout")
)

// Info groups information about fresh new process
type Info struct {
	PID     int
	Process *os.Process
	Stdin   io.WriteCloser
}

// Option is an option func to change the underlying command
type Option func(c *exec.Cmd)

// Start starts a new process
// Returns:
// - network address of child process
// - process id
// - error
func Start(logger *logger.Logger, path string, config *Config, uid, gid int, args []string, opts ...Option) (proc *Info, err error) {
	return StartContext(nil, logger, path, config, uid, gid, args, opts...)
}

// StartContext starts a new process with context.
// Returns:
// - network address of child process
// - process id
// - error
func StartContext(ctx context.Context, logger *logger.Logger, path string, config *Config, uid, gid int, args []string, opts ...Option) (*Info, error) {
	cmd := getCmd(ctx, logger, path, []string{}, uid, gid, args...)
	for _, o := range opts {
		o(cmd)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	// start process
	if err := cmd.Start(); err != nil {
		return nil, errors.New(err, fmt.Sprintf("failed to start '%s'", path))
	}

	// Hook to JobObject on windows, noop on other platforms.
	// This ties the application processes lifespan to the agent's.
	// Fixes the orphaned beats processes left behind situation
	// after the agent process gets killed.
	if err := proc.JobObject.Assign(cmd.Process); err != nil {
		logger.Errorf("application process failed job assign: %v", err)
	}

	return &Info{
		PID:     cmd.Process.Pid,
		Process: cmd.Process,
		Stdin:   stdin,
	}, err
}

// Stop stops the process cleanly.
func (i *Info) Stop() error {
	return terminateCmd(i.Process)
}

// StopWait stops the process and waits for it to exit.
func (i *Info) StopWait() error {
	err := i.Stop()
	if err != nil {
		return err
	}
	_, err = i.Process.Wait()
	return err
}
