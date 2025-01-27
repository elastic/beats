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

package integration

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

var (
	compiling sync.Mutex
)

// RunningBeat describes the running Beat binary.
type RunningBeat struct {
	c           *exec.Cmd
	outputRW    sync.RWMutex
	output      []string
	outputDone  chan struct{}
	watcher     OutputWatcher
	keepRunning bool
}

// CollectOutput returns the last `limit` lines of the currently
// accumulated output.
// `limit=-1` returns the entire output from the beginning.
func (b *RunningBeat) CollectOutput(limit int) string {
	b.outputRW.RLock()
	defer b.outputRW.RUnlock()
	if limit < 0 {
		limit = len(b.output)
	}

	builder := strings.Builder{}
	output := b.output
	if len(output) > limit {
		output = output[len(output)-limit:]
	}

	m := make(map[string]any)
	for i, l := range output {
		err := json.Unmarshal([]byte(l), &m)
		if err != nil {
			builder.WriteString(l)
		} else {
			pretty, _ := json.MarshalIndent(m, "", "  ")
			builder.Write(pretty)
		}
		if i < len(output)-1 {
			builder.WriteByte('\n')
		}
	}

	return builder.String()
}

// Wait until the Beat exists and all the output is processed
func (b *RunningBeat) Wait() error {
	err := b.c.Wait()
	<-b.outputDone
	return err
}

func (b *RunningBeat) writeOutputLine(line string) {
	b.outputRW.Lock()
	defer b.outputRW.Unlock()

	b.output = append(b.output, line)

	if b.watcher == nil {
		return
	}

	b.watcher.Inspect(line)
	if b.watcher.Observed() {
		if !b.keepRunning {
			_ = b.c.Process.Kill()
		}
		b.watcher = nil
	}
}

// RunBeatOptions describes the options for running a Beat
type RunBeatOptions struct {
	// Beatname, for example "filebeat".
	Beatname string
	// Config for the Beat written in YAML
	Config string
	// Args sets additional arguments to pass when running the binary.
	Args []string
	// KeepRunning if set to `true` observing all
	// the expected output would not kill the process.
	//
	// In this case user controls the runtime through the context
	// passed in `RunBeat`.
	KeepRunning bool
}

// Runs a Beat binary with the given config and args.
// Returns a `RunningBeat` that allow to collect the output and wait until the exit.
func RunBeat(ctx context.Context, t *testing.T, opts RunBeatOptions, watcher OutputWatcher) *RunningBeat {
	t.Logf("preparing to run %s...", opts.Beatname)

	binaryFilename := findBeatBinaryPath(t, opts.Beatname)

	// create a temporary Beat config
	cfgPath := filepath.Join(t.TempDir(), fmt.Sprintf("%s.yml", opts.Beatname))
	homePath := filepath.Join(t.TempDir(), "home")

	err := os.WriteFile(cfgPath, []byte(opts.Config), 0777)
	if err != nil {
		t.Fatalf("failed to create a temporary config file: %s", err)
		return nil
	}
	t.Logf("temporary config has been created at %s", cfgPath)

	// compute the args for execution
	baseArgs := []string{
		// logging to stderr instead of log files
		"-e",
		"-c", cfgPath,
		// we want all the logs
		"-E", "logging.level=debug",
		// so we can run multiple Beats at the same time
		"--path.home", homePath,
	}
	execArgs := make([]string, 0, len(baseArgs)+len(opts.Args))
	execArgs = append(execArgs, baseArgs...)
	execArgs = append(execArgs, opts.Args...)

	t.Logf("running %s %s", binaryFilename, strings.Join(execArgs, " "))
	c := exec.CommandContext(ctx, binaryFilename, execArgs...)

	output, err := c.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to create the stdout pipe: %s", err)
		return nil
	}
	c.Stderr = c.Stdout

	b := &RunningBeat{
		c:           c,
		watcher:     watcher,
		keepRunning: opts.KeepRunning,
		outputDone:  make(chan struct{}),
	}

	go func() {
		scanner := bufio.NewScanner(output)
		for scanner.Scan() {
			b.writeOutputLine(scanner.Text())
		}
		if scanner.Err() != nil {
			t.Logf("error while reading from stdout/stderr: %s", scanner.Err())
		}
		close(b.outputDone)
	}()

	err = c.Start()
	if err != nil {
		t.Fatalf("failed to start Filebeat command: %s", err)
		return nil
	}

	t.Logf("%s is running (pid: %d)", binaryFilename, c.Process.Pid)

	return b
}

// EnsureCompiled ensures that the given Beat is compiled and ready
// to run.
func EnsureCompiled(ctx context.Context, t *testing.T, beatname string) (path string) {
	compiling.Lock()
	defer compiling.Unlock()

	t.Logf("ensuring the %s binary is available...", beatname)

	binaryFilename := findBeatBinaryPath(t, beatname)
	_, err := os.Stat(binaryFilename)
	if err == nil {
		t.Logf("found existing %s binary at %s", beatname, binaryFilename)
		return binaryFilename
	}

	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed to check for compiled binary %s: %s", binaryFilename, err)
		return ""
	}

	mageCommand := "mage"
	if runtime.GOOS == "windows" {
		mageCommand += ".exe"
	}
	args := []string{"build"}
	t.Logf("existing %s binary not found, building with \"%s %s\"... ", mageCommand, binaryFilename, strings.Join(args, " "))
	c := exec.CommandContext(ctx, mageCommand, args...)
	c.Dir = filepath.Dir(binaryFilename)
	output, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build %s binary: %s\n%s", beatname, err, output)
		return ""
	}

	_, err = os.Stat(binaryFilename)
	if err == nil {
		t.Logf("%s binary has been successfully built ", binaryFilename)
		return binaryFilename
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("building command for binary %s succeeded but the binary was not created: %s", binaryFilename, err)
		return ""
	}

	return ""
}

func findBeatDir(t *testing.T, beatName string) string {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get the working directory: %s", err)
		return ""
	}
	t.Logf("searching for the %s directory, starting with %s...", beatName, pwd)
	for pwd != "" {
		stat, err := os.Stat(filepath.Join(pwd, beatName))
		if errors.Is(err, os.ErrNotExist) || !stat.IsDir() {
			pwd = filepath.Dir(pwd)
			continue
		}
		return filepath.Join(pwd, beatName)
	}
	t.Fatalf("could not find the %s base directory", beatName)
	return ""
}

func findBeatBinaryPath(t *testing.T, beatname string) string {
	baseDir := findBeatDir(t, beatname)
	t.Logf("found %s directory at %s", beatname, baseDir)
	binary := filepath.Join(baseDir, beatname)
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	return binary
}
