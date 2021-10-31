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

package xbuild

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/urso/magetools/clitool"
)

// DockerImage provides based on downloadable docker images.
type DockerImage struct {
	Image   string
	Workdir string
	Volumes map[string]string
	Env     map[string]string
}

type DockerRunner struct {
	Image   *DockerImage
	Keep    bool // keep container
	Verbose bool
}

// Build pulls the required image.
func (p *DockerImage) Start() error {
	cli := clitool.NewCLIExecutor(false)
	_, err := cli.Exec(
		context.Background(),
		clitool.Command{
			Path:       "docker",
			SubCommand: []string{"pull"},
		},
		clitool.CreateArgs(
			clitool.Positional(p.Image),
		),
		os.Stdout,
		os.Stderr,
	)
	return err
}

func (p *DockerImage) Stop() error {
	return nil
}

// Executor creates the execution environment
func (p *DockerImage) Executor(verbose bool) (clitool.Executor, error) {
	return &DockerRunner{
		Image:   p,
		Verbose: verbose,
	}, nil
}

func (p *DockerImage) Shell() error {
	e, _ := p.Executor(false)
	_, err := e.Exec(context.Background(),
		clitool.Command{Path: "bash"}, clitool.CreateArgs(),
		os.Stdout, os.Stderr,
	)
	return err
}

func (dr *DockerRunner) Exec(
	ctx context.Context,
	cmd clitool.Command,
	args *clitool.Args,
	stdout, stderr io.Writer,
) (bool, error) {
	var dockerArgs = []clitool.ArgOpt{
		clitool.BoolFlag("--rm", !dr.Keep),
		clitool.Flag("-i", ""),
		clitool.Flag("-t", ""),
		clitool.Positional(dr.Image.Image),
	}

	for k, v := range dr.Image.Env {
		dockerArgs = append(dockerArgs, clitool.Flag("-e", fmt.Sprintf("%v=%v", k, v)))
	}

	for k, v := range dr.Image.Volumes {
		dockerArgs = append(dockerArgs, clitool.Flag("-v", fmt.Sprintf("%v:%v", k, v)))
	}

	w := cmd.WorkingDir
	if w == "" {
		w = dr.Image.Workdir
	}
	if w != "" {
		dockerArgs = append(dockerArgs, clitool.Flag("-w", w))
	}

	arguments := args.Build()
	if len(cmd.SubCommand) > 0 {
		tmp := make([]string, 0, len(arguments)+len(cmd.SubCommand))
		tmp = append(tmp, cmd.SubCommand...)
		tmp = append(tmp, arguments...)
		arguments = tmp
	}

	runArgs := append([]string{"run"}, clitool.CreateArgs(dockerArgs...).Build()...)
	runArgs = append(runArgs, cmd.Path)
	runArgs = append(runArgs, arguments...)

	osCommand := exec.CommandContext(ctx, "docker", runArgs...)
	osCommand.Stdout = stdout
	osCommand.Stderr = stderr
	osCommand.Stdin = os.Stdin

	if dr.Verbose {
		fmt.Printf("Exec docker %v\n", runArgs)
	}

	didRun, exitCode, err := checkError(osCommand.Run())
	if err == nil {
		return didRun, nil
	}

	if dr.Verbose {
		fmt.Println("  => exit code:", exitCode)
	}

	if !didRun {
		return false, fmt.Errorf("failed to run command: %+v", err)
	}
	return true, fmt.Errorf("command %v failed with %v: %+v", cmd.Path, exitCode, err)
}

func checkError(err error) (bool, int, error) {
	if err == nil {
		return true, 0, nil
	}

	switch e := err.(type) {
	case *exec.ExitError:
		return e.Exited(), exitStatus(err), err
	case interface{ ExitStatus() int }:
		return false, exitStatus(err), err
	default:
		return false, 1, err
	}
}

func exitStatus(err error) int {
	type exitStatus interface {
		ExitStatus() int
	}

	if err == nil {
		return 0
	}

	switch e := err.(type) {
	case exitStatus:
		return e.ExitStatus()
	case *exec.ExitError:
		if sysErr, ok := e.Sys().(exitStatus); ok {
			return sysErr.ExitStatus()
		}
	}

	return 1
}
