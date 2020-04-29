package clitool

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// CLIExecutor runs a command with arguments using the os.Exec function.
type CLIExecutor struct {
	verbose bool
}

func NewCLIExecutor(verbose bool) *CLIExecutor {
	return &CLIExecutor{verbose: verbose}
}

func (e *CLIExecutor) ExecCollectOutput(
	ctx context.Context,
	c Command,
	args *Args,
) (string, error) {
	var buf strings.Builder

	_, err := e.Exec(ctx, c, args, &buf, os.Stderr)
	return buf.String(), err
}

func (e *CLIExecutor) Exec(
	ctx context.Context,
	c Command,
	args *Args,
	stdout, stderr io.Writer,
) (bool, error) {
	command := c.Path
	if command == "" {
		return false, errors.New("No command configured")
	}

	command = os.Expand(command, func(s string) string {
		if tmp, ok := args.Environment[s]; ok {
			return tmp
		}
		return os.Getenv(s)
	})

	env := os.Environ()
	for k, v := range args.Environment {
		env = append(env, k+"="+v)
	}

	arguments := args.Build()
	if len(c.SubCommand) > 0 {
		tmp := make([]string, 0, len(arguments)+len(c.SubCommand))
		tmp = append(tmp, c.SubCommand...)
		tmp = append(tmp, arguments...)
		arguments = tmp
	}

	osCommand := exec.CommandContext(ctx, c.Path, arguments...)
	osCommand.Env = env
	osCommand.Dir = c.WorkingDir
	osCommand.Stdout = stdout
	osCommand.Stderr = stderr
	osCommand.Stdin = os.Stdin

	if e.verbose {
		fmt.Printf("Exec (working dir '%v'): `%v %v`\n",
			c.WorkingDir,
			command,
			strings.Join(arguments, " "))
	}

	didRun, exitCode, err := checkError(osCommand.Run())
	if err == nil {
		return didRun, nil
	}

	if e.verbose {
		fmt.Println("  => exit code:", exitCode)
	}

	if !didRun {
		return false, fmt.Errorf("failed to run command: %+v", err)
	}
	return true, fmt.Errorf("command %v failed with %v: %+v", command, exitCode, err)
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
