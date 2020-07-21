// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package process

import (
	"fmt"
	"io"
	"os"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
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

// Start starts a new process
// Returns:
// - network address of child process
// - process id
// - error
func Start(logger *logger.Logger, path string, config *Config, uid, gid int, arg ...string) (proc *Info, err error) {
	cmd := getCmd(logger, path, []string{}, uid, gid, arg...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	// start process
	if err := cmd.Start(); err != nil {
		return nil, errors.New(err, fmt.Sprintf("failed to start '%s'", path))
	}

	return &Info{
		PID:     cmd.Process.Pid,
		Process: cmd.Process,
		Stdin:   stdin,
	}, err
}
