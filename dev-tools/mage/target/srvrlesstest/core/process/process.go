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

package process

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Info groups information about fresh new process
type Info struct {
	PID     int
	Process *os.Process
	Stdin   io.WriteCloser
	Stderr  io.ReadCloser
}

// CmdOption is an option func to change the underlying command
type CmdOption func(c *exec.Cmd) error

// StartConfig configuration for the process start set by the StartOption functions
type StartConfig struct {
	ctx       context.Context
	uid, gid  int
	args, env []string
	cmdOpts   []CmdOption
}

// StartOption start options function
type StartOption func(cfg *StartConfig)

// Start starts a new process
func Start(path string, opts ...StartOption) (proc *Info, err error) {
	// Apply options
	c := StartConfig{
		uid: os.Geteuid(),
		gid: os.Getegid(),
	}

	for _, opt := range opts {
		opt(&c)
	}

	return startContext(c.ctx, path, c.uid, c.gid, c.args, c.env, c.cmdOpts...)
}

// WithContext sets an optional context
func WithContext(ctx context.Context) StartOption {
	return func(cfg *StartConfig) {
		cfg.ctx = ctx
	}
}

// WithArgs sets arguments
func WithArgs(args []string) StartOption {
	return func(cfg *StartConfig) {
		cfg.args = args
	}
}

// WithEnv sets the environment variables
func WithEnv(env []string) StartOption {
	return func(cfg *StartConfig) {
		cfg.env = env
	}
}

// WithUID sets UID
func WithUID(uid int) StartOption {
	return func(cfg *StartConfig) {
		cfg.uid = uid
	}
}

// WithGID sets GID
func WithGID(gid int) StartOption {
	return func(cfg *StartConfig) {
		cfg.gid = gid
	}
}

// WithCmdOptions sets the exec.Cmd options
func WithCmdOptions(cmdOpts ...CmdOption) StartOption {
	return func(cfg *StartConfig) {
		cfg.cmdOpts = cmdOpts
	}
}

// WithWorkDir sets the cmd working directory
func WithWorkDir(wd string) CmdOption {
	return func(c *exec.Cmd) error {
		c.Dir = wd
		return nil
	}
}

// Kill kills the process.
func (i *Info) Kill() error {
	return killCmd(i.Process)
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

// Wait returns a channel that will send process state once it exits. Each
// call to Wait() creates a goroutine. Failure to read from the returned
// channel will leak this goroutine.
func (i *Info) Wait() <-chan *os.ProcessState {
	ch := make(chan *os.ProcessState)

	go func() {
		procState, err := i.Process.Wait()
		if err != nil {
			// process is not a child - some OSs requires process to be child
			externalProcess(i.Process)
		}
		ch <- procState
	}()

	return ch
}

// startContext starts a new process with context. The context is optional and can be nil.
func startContext(ctx context.Context, path string, uid, gid int, args []string, env []string, opts ...CmdOption) (*Info, error) {
	cmd, err := getCmd(ctx, path, env, uid, gid, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create command for %q: %w", path, err)
	}
	for _, o := range opts {
		if err := o(cmd); err != nil {
			return nil, fmt.Errorf("failed to set option command for %q: %w", path, err)
		}
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin for %q: %w", path, err)
	}

	var stderr io.ReadCloser
	if cmd.Stderr == nil {
		stderr, err = cmd.StderrPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to create stderr for %q: %w", path, err)
		}
	}

	// start process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start %q: %w", path, err)
	}

	// Hook to JobObject on windows, noop on other platforms.
	// This ties the application processes lifespan to the agent's.
	// Fixes the orphaned beats processes left behind situation
	// after the agent process gets killed.
	if err := JobObject.Assign(cmd.Process); err != nil {
		_ = killCmd(cmd.Process)
		return nil, fmt.Errorf("failed job assignment %q: %w", path, err)
	}

	return &Info{
		PID:     cmd.Process.Pid,
		Process: cmd.Process,
		Stdin:   stdin,
		Stderr:  stderr,
	}, err
}
