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
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	// map of Beat names to binary hashes that `EnsureCompiled` function built
	compiled = map[string]string{}
	hash     = sha256.New()
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

// RunBeat runs a Beat binary with the given config and args.
// Returns a `RunningBeat` that allow to collect the output and wait until the exit.
func RunBeat(ctx context.Context, t *testing.T, opts RunBeatOptions, watcher OutputWatcher) *RunningBeat {
	t.Logf("preparing to run %s...", opts.Beatname)

	binaryFilename := findBeatBinaryPath(t, opts.Beatname)
	dir := t.TempDir()
	// create a temporary Beat config
	cfgPath := filepath.Join(dir, fmt.Sprintf("%s.yml", opts.Beatname))
	homePath := filepath.Join(dir, "home")

	err := os.WriteFile(cfgPath, []byte(opts.Config), 0644)
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

	// we must use 2 pipes since writes are not aligned by lines
	// part of the stdout output can end up in the middle of the stderr line
	stdout, err := c.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to create the stdout pipe: %s", err)
		return nil
	}

	stderr, err := c.StderrPipe()
	if err != nil {
		t.Fatalf("failed to create the stdout pipe: %s", err)
		return nil
	}

	b := &RunningBeat{
		c:           c,
		watcher:     watcher,
		keepRunning: opts.KeepRunning,
		outputDone:  make(chan struct{}),
	}

	var wg sync.WaitGroup
	// arbitrary buffer size
	output := make(chan string, 128)

	wg.Add(2)
	go func() {
		processPipe(t, stdout, output)
		wg.Done()
	}()
	go func() {
		processPipe(t, stderr, output)
		wg.Done()
	}()
	go func() {
		wg.Wait()
		close(output)
	}()
	go func() {
		for line := range output {
			b.writeOutputLine(line)
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

func processPipe(t *testing.T, r io.Reader, output chan<- string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		output <- scanner.Text()
	}
	if scanner.Err() != nil {
		t.Logf("error while reading from stdout/stderr: %s", scanner.Err())
	}
}

// EnsureCompiled ensures that the given Beat is compiled and ready
// to run.
// This functions allows to use binaries only built by this function.
// Externally created binaries will be removed and rebuilt.
func EnsureCompiled(ctx context.Context, t *testing.T, beatname string) (path string) {
	compiling.Lock()
	defer compiling.Unlock()

	t.Logf("ensuring the %s binary is available...", beatname)
	binaryFilename := findBeatBinaryPath(t, beatname)
	// empty if the binary was not compiled before
	expectedHash := compiled[beatname]
	// we allow to use binaries only built by this function.
	// binaries from different origins are marked as outdated
	_, err := os.Stat(binaryFilename)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed to check for compiled binary %s: %s", binaryFilename, err)
		return ""
	}
	if err == nil {
		actualHash := hashBinary(t, binaryFilename)
		if actualHash == expectedHash {
			t.Logf("%s binary has been compiled before at %s, using...", beatname, binaryFilename)
			return binaryFilename
		}
		t.Logf("found outdated %s binary at %s, removing...", beatname, binaryFilename)
		err := os.Remove(binaryFilename)
		if err != nil {
			t.Fatalf("failed to remove outdated %s binary at %s: %s", beatname, binaryFilename, err)
			return ""
		}
	} else {
		t.Logf("%s binary was not found at %s", beatname, binaryFilename)
	}

	mageCommand := "mage"
	if runtime.GOOS == "windows" {
		mageCommand += ".exe"
	}
	args := []string{"build"}
	t.Logf("building %s binary with \"%s %s\"... ", binaryFilename, mageCommand, strings.Join(args, " "))
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
		compiled[beatname] = hashBinary(t, binaryFilename)
		return binaryFilename
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("building command for binary %s succeeded but the binary was not created: %s", binaryFilename, err)
		return ""
	}

	return ""
}

func hashBinary(t *testing.T, filename string) string {
	f, err := os.Open(filename)
	if err != nil {
		t.Fatalf("failed to open %s: %s", filename, err)
		return ""
	}
	defer f.Close()
	hash.Reset()
	if _, err := io.Copy(hash, f); err != nil {
		t.Fatalf("failed to hash %s: %s", filename, err)
		return ""
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
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
