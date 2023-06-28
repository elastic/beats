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

//go:build integration

package integration

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
)

type BeatProc struct {
	Binary        string
	Args          []string
	Cmd           *exec.Cmd
	t             *testing.T
	tempDir       string
	configFile    string
	beatName      string
	logFileOffset int64
	stdout        *os.File
	stderr        *os.File
	Process       *os.Process
}

type Meta struct {
	UUID       uuid.UUID `json:"uuid"`
	FirstStart time.Time `json:"first_start"`
}

// NewBeat createa a new Beat process from the system tests binary.
// It sets some required options like the home path, logging, etc.
// `tempDir` will be used as home and logs directory for the Beat
// `args` will be passed as CLI arguments to the Beat
func NewBeat(t *testing.T, beatName, binary string, args ...string) BeatProc {
	require.FileExistsf(t, binary, "beat binary must exists")
	tempDir := createTempDir(t)
	configFile := filepath.Join(tempDir, beatName+".yml")
	stdoutFile, err := os.Create(filepath.Join(tempDir, "stdout"))
	require.NoError(t, err, "error creating stdout file")
	stderrFile, err := os.Create(filepath.Join(tempDir, "stderr"))
	require.NoError(t, err, "error creating stderr file")
	p := BeatProc{
		Binary: binary,
		Args: append([]string{
			beatName,
			"--systemTest",
			"--path.home", tempDir,
			"--path.logs", tempDir,
			"-E", "logging.to_files=true",
			"-E", "logging.files.rotateeverybytes=104857600", // About 100MB
		}, args...),
		tempDir:    tempDir,
		beatName:   beatName,
		configFile: configFile,
		t:          t,
		stdout:     stdoutFile,
		stderr:     stderrFile,
	}
	return p
}

// Start starts the Beat process
// args are extra arguments to be passed to the Beat
func (b *BeatProc) Start(args ...string) {
	t := b.t
	allArgs := append(b.Args, args...)
	fullPath, err := filepath.Abs(b.Binary)
	if err != nil {
		t.Fatalf("could not get full path from %q, err: %s", b.Binary, err)
	}

	b.stdout.Seek(0, 0)
	b.stdout.Truncate(0)
	b.stderr.Seek(0, 0)
	b.stderr.Truncate(0)
	var procAttr os.ProcAttr
	procAttr.Files = []*os.File{os.Stdin, b.stdout, b.stderr}
	process, err := os.StartProcess(fullPath, allArgs, &procAttr)
	require.NoError(t, err, "error starting beat process")
	b.Process = process
	t.Cleanup(func() {
		if err := b.Process.Signal(os.Interrupt); err != nil {
			if errors.Is(err, os.ErrProcessDone) {
				return
			}
			t.Fatalf("could not stop process with PID: %d, err: %s", b.Process.Pid, err)
		}
	})
}

// Stop stops the Beat process
// Start adds Cleanup function to stop when test ends, only run this if you want to inspect logs after beat shutsdown
func (b *BeatProc) Stop() {
	if err := b.Process.Signal(os.Interrupt); err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			return
		}
		b.t.Fatalf("could not stop process with PID: %d, err: %s", b.Process.Pid, err)
	}
}

// LogContains looks for `s` as a substring of every log line,
// it will open the log file on every call, read it until EOF,
// then close it.
func (b *BeatProc) LogContains(s string) string {
	t := b.t
	logFile := b.openLogFile()
	_, err := logFile.Seek(b.logFileOffset, os.SEEK_SET)
	if err != nil {
		t.Fatalf("could not set offset for '%s': %s", logFile.Name(), err)
	}

	defer func() {
		offset, err := logFile.Seek(0, os.SEEK_CUR)
		if err != nil {
			t.Fatalf("could not read offset for '%s': %s", logFile.Name(), err)
		}
		b.logFileOffset = offset
		if err := logFile.Close(); err != nil {
			// That's not quite a test error, but it can impact
			// next executions of LogContains, so treat it as an error
			t.Errorf("could not close log file: %s", err)
		}
	}()

	r := bufio.NewReader(logFile)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				t.Fatalf("error reading log file '%s': %s", logFile.Name(), err)
			}
			break
		}
		if strings.Contains(line, s) {
			return line
		}
	}

	return ""
}

// WaitForLogs waits for the specified string s to be present in the logs within
// the given timeout duration and fails the test if s is not found.
// msgAndArgs should be a format string and arguments that will be printed
// if the logs are not found, providing additional context for debugging.
func (b *BeatProc) WaitForLogs(s string, timeout time.Duration, msgAndArgs ...any) string {
	b.t.Helper()
	var returnValue string
	require.Eventually(b.t, func() bool {
		returnValue = b.LogContains(s)
		return returnValue != ""
	}, timeout, 100*time.Millisecond, msgAndArgs...)

	return returnValue
}

// TempDir returns the temporary directory
// used by that Beat, on a successful test,
// the directory is automatically removed.
// On failure, the temporary directory is kept.
func (b *BeatProc) TempDir() string {
	return b.tempDir
}

// WriteConfigFile writes the provided configuration string cfg to a file.
// This file will be used as the configuration file for the Beat.
func (b *BeatProc) WriteConfigFile(cfg string) {
	if err := os.WriteFile(b.configFile, []byte(cfg), 0o644); err != nil {
		b.t.Fatalf("cannot create config file '%s': %s", b.configFile, err)
	}

	b.Args = append(b.Args, "-c", b.configFile)
}

// openLogFile opens the log file for reading and returns it.
// It also registers a cleanup function to close the file
// when the test ends.
func (b *BeatProc) openLogFile() *os.File {
	t := b.t
	glob := fmt.Sprintf("%s-*.ndjson", filepath.Join(b.tempDir, b.beatName))
	files, err := filepath.Glob(glob)
	if err != nil {
		t.Fatalf("could not expand log file glob: %s", err)
	}

	require.Eventually(t, func() bool {
		files, err = filepath.Glob(glob)
		if err != nil {
			t.Fatalf("could not expand log file glob: %s", err)
		}
		return len(files) == 1
	}, 5*time.Second, 100*time.Millisecond,
		"waiting for log file matching glob '%s' to be created", glob)

	// On a normal operation there must be a single log, if there are more
	// than one, then there is an issue and the Beat is logging too much,
	// which is enough to stop the test
	if len(files) != 1 {
		t.Fatalf("there must be only one log file for %s, found: %d",
			glob, len(files))
	}

	f, err := os.Open(files[0])
	if err != nil {
		t.Fatalf("could not open log file '%s': %s", files[0], err)
	}

	return f
}

// createTempDir creates a temporary directory that will be
// removed after the tests passes.
//
// If the test fails, the temporary directory is not removed.
//
// If the tests are run with -v, the temporary directory will
// be logged.
func createTempDir(t *testing.T) string {
	tempDir, err := filepath.Abs(filepath.Join("../../build/integration-tests/",
		fmt.Sprintf("%s-%d", t.Name(), time.Now().Unix())))
	if err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(tempDir, 0o766); err != nil {
		t.Fatalf("cannot create tmp dir: %s, msg: %s", err, err.Error())
	}

	cleanup := func() {
		if !t.Failed() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Errorf("could not remove temp dir '%s': %s", tempDir, err)
			}
		} else {
			t.Logf("Temporary directory saved: %s", tempDir)
		}
	}
	t.Cleanup(cleanup)

	return tempDir
}

// EnsureESIsRunning ensures Elasticsearch is running and is reachable
// using the default test credentials or the corresponding environment
// variables.
func EnsureESIsRunning(t *testing.T) {
	t.Helper()

	esHost := os.Getenv("ES_HOST")
	if esHost == "" {
		esHost = "localhost"
	}

	esPort := os.Getenv("ES_PORT")
	if esPort == "" {
		esPort = "9200"
	}

	esURL := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%s", esHost, esPort),
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(500*time.Second))
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, esURL.String(), nil)
	if err != nil {
		t.Fatalf("cannot create request to ensure ES is running: %s", err)
	}

	user := os.Getenv("ES_USER")
	if user == "" {
		user = "admin"
	}

	pass := os.Getenv("ES_PASS")
	if pass == "" {
		pass = "testing"
	}

	req.SetBasicAuth(user, pass)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// If you're reading this message, you probably forgot to start ES
		// run `mage compose:Up` from Filebeat's folder to start all
		// containers required for integration tests
		t.Fatalf("cannot execute HTTP request to ES: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected HTTP status: %d, expecting 200 - OK", resp.StatusCode)
	}
}

func (b *BeatProc) FileContains(filename string, match string) string {
	file, err := os.Open(filename)
	require.NoErrorf(b.t, err, "error opening: %s", filename)
	r := bufio.NewReader(file)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				b.t.Fatalf("error reading log file '%s': %s", file.Name(), err)
			}
			break
		}
		if strings.Contains(line, match) {
			return line
		}
	}
	return ""
}

func (b *BeatProc) WaitFileContains(filename string, match string, waitFor time.Duration) string {
	var returnValue string
	require.Eventuallyf(b.t,
		func() bool {
			returnValue = b.FileContains(filename, match)
			return returnValue != ""
		}, waitFor, 100*time.Millisecond, "match string '%s' not found in %s", match, filename)

	return returnValue
}

func (b *BeatProc) WaitStdErrContains(match string, waitFor time.Duration) string {
	return b.WaitFileContains(b.stderr.Name(), match, waitFor)
}

func (b *BeatProc) WaitStdOutContains(match string, waitFor time.Duration) string {
	return b.WaitFileContains(b.stdout.Name(), match, waitFor)
}

func (b *BeatProc) LoadMeta() (Meta, error) {
	m := Meta{}
	metaFile, err := os.Open(filepath.Join(b.TempDir(), "data", "meta.json"))
	if err != nil {
		return m, err
	}
	defer metaFile.Close()

	metaBytes, _ := ioutil.ReadAll(metaFile)
	json.Unmarshal(metaBytes, &m)
	return m, nil
}

func GetESURL(t *testing.T, scheme string) url.URL {
	t.Helper()

	esHost := os.Getenv("ES_HOST")
	if esHost == "" {
		esHost = "localhost"
	}

	esPort := os.Getenv("ES_PORT")
	if esPort == "" {
		switch scheme {
		case "http":
			esPort = "9200"
		case "https":
			esPort = "9201"
		}
	}

	esURL := url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%s", esHost, esPort),
	}
	return esURL
}
