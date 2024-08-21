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
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
)

type BeatProc struct {
	Args                []string
	baseArgs            []string
	Binary              string
	RestartOnBeatOnExit bool
	beatName            string
	cmdMutex            sync.Mutex
	configFile          string
	fullPath            string
	logFileOffset       int64
	eventLogFileOffset  int64
	t                   *testing.T
	tempDir             string
	stdin               io.WriteCloser
	stdout              *os.File
	stderr              *os.File
	Process             *os.Process
}

type Meta struct {
	UUID       uuid.UUID `json:"uuid"`
	FirstStart time.Time `json:"first_start"`
}

type IndexTemplateResult struct {
	IndexTemplates []IndexTemplateEntry `json:"index_templates"`
}

type IndexTemplateEntry struct {
	Name          string        `json:"name"`
	IndexTemplate IndexTemplate `json:"index_template"`
}

type IndexTemplate struct {
	IndexPatterns []string `json:"index_patterns"`
}

type SearchResult struct {
	Hits Hits `json:"hits"`
}

type Hits struct {
	Total Total `json:"total"`
}

type Total struct {
	Value int `json:"value"`
}

// NewBeat creates a new Beat process from the system tests binary.
// It sets some required options like the home path, logging, etc.
// `tempDir` will be used as home and logs directory for the Beat
// `args` will be passed as CLI arguments to the Beat
func NewBeat(t *testing.T, beatName, binary string, args ...string) *BeatProc {
	require.FileExistsf(t, binary, "beat binary must exists")
	tempDir := createTempDir(t)
	configFile := filepath.Join(tempDir, beatName+".yml")

	stdoutFile, err := os.Create(filepath.Join(tempDir, "stdout"))
	require.NoError(t, err, "error creating stdout file")
	stderrFile, err := os.Create(filepath.Join(tempDir, "stderr"))
	require.NoError(t, err, "error creating stderr file")

	p := BeatProc{
		Binary: binary,
		baseArgs: append([]string{
			beatName,
			"--systemTest",
			"--path.home", tempDir,
			"--path.logs", tempDir,
			"-E", "logging.to_files=true",
			"-E", "logging.files.rotateeverybytes=104857600", // About 100MB
			"-E", "logging.files.rotateonstartup=false",
		}, args...),
		tempDir:    tempDir,
		beatName:   beatName,
		configFile: configFile,
		t:          t,
		stdout:     stdoutFile,
		stderr:     stderrFile,
	}
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}
		reportErrors(t, tempDir, beatName)
	})
	return &p
}

// NewAgentBeat creates a new agentbeat process that runs the beatName as a subcommand.
// See `NewBeat` for options and information for the parameters.
func NewAgentBeat(t *testing.T, beatName, binary string, args ...string) *BeatProc {
	require.FileExistsf(t, binary, "agentbeat binary must exists")
	tempDir := createTempDir(t)
	configFile := filepath.Join(tempDir, beatName+".yml")

	stdoutFile, err := os.Create(filepath.Join(tempDir, "stdout"))
	require.NoError(t, err, "error creating stdout file")
	stderrFile, err := os.Create(filepath.Join(tempDir, "stderr"))
	require.NoError(t, err, "error creating stderr file")

	p := BeatProc{
		Binary: binary,
		baseArgs: append([]string{
			"agentbeat",
			"--systemTest",
			beatName,
			"--path.home", tempDir,
			"--path.logs", tempDir,
			"-E", "logging.to_files=true",
			"-E", "logging.files.rotateeverybytes=104857600", // About 100MB
			"-E", "logging.files.rotateonstartup=false",
		}, args...),
		tempDir:    tempDir,
		beatName:   beatName,
		configFile: configFile,
		t:          t,
		stdout:     stdoutFile,
		stderr:     stderrFile,
	}
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}
		reportErrors(t, tempDir, beatName)
	})
	return &p
}

// Start starts the Beat process
// args are extra arguments to be passed to the Beat.
func (b *BeatProc) Start(args ...string) {
	t := b.t
	fullPath, err := filepath.Abs(b.Binary)
	if err != nil {
		t.Fatalf("could not get full path from %q, err: %s", b.Binary, err)
	}

	b.fullPath = fullPath
	b.Args = append(b.baseArgs, args...)

	done := atomic.MakeBool(false)
	wg := sync.WaitGroup{}
	if b.RestartOnBeatOnExit {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !done.Load() {
				b.startBeat()
				b.waitBeatToExit()
			}
		}()
	} else {
		b.startBeat()
	}

	t.Cleanup(func() {
		b.cmdMutex.Lock()
		// 1. Kill the Beat
		if err := b.Process.Signal(os.Interrupt); err != nil {
			if !errors.Is(err, os.ErrProcessDone) {
				t.Fatalf("could not stop process with PID: %d, err: %s",
					b.Process.Pid, err)
			}
		}

		// Make sure the goroutine restarting the Beat has exited
		if b.RestartOnBeatOnExit {
			// 2. Set the done flag so the goroutine loop can exit
			done.Store(true)
			// 3. Release the mutex, keeping it locked
			// until now ensures a new process won't
			// start.  Lock must be released before
			// wg.Wait() or there is a possibility of
			// deadlock.
			b.cmdMutex.Unlock()
			// 4. Wait for the goroutine to finish, this helps ensuring
			// no other Beat process was started
			wg.Wait()
		} else {
			b.cmdMutex.Unlock()
		}
	})
}

// startBeat starts the Beat process. This method
// does not block nor waits the Beat to finish.
func (b *BeatProc) startBeat() {
	b.cmdMutex.Lock()
	defer b.cmdMutex.Unlock()

	_, _ = b.stdout.Seek(0, 0)
	_ = b.stdout.Truncate(0)
	_, _ = b.stderr.Seek(0, 0)
	_ = b.stderr.Truncate(0)

	cmd := exec.Cmd{
		Path:   b.fullPath,
		Args:   b.Args,
		Stdout: b.stdout,
		Stderr: b.stderr,
	}

	var err error
	b.stdin, err = cmd.StdinPipe()
	require.NoError(b.t, err, "could not get cmd StdinPipe")

	err = cmd.Start()
	require.NoError(b.t, err, "error starting beat process")

	b.Process = cmd.Process
}

// waitBeatToExit blocks until the Beat exits, it returns
// the process' exit code.
// `startBeat` must be called before this method.
func (b *BeatProc) waitBeatToExit() int {
	processState, err := b.Process.Wait()
	if err != nil {
		b.t.Fatalf("error waiting for %q to finish: %s. Exit code: %d",
			b.beatName, err, processState.ExitCode())
	}

	return processState.ExitCode()
}

// Stop stops the Beat process
// Start adds Cleanup function to stop when test ends, only run this if you want to inspect logs after beat shutsdown
func (b *BeatProc) Stop() {
	b.cmdMutex.Lock()
	defer b.cmdMutex.Unlock()
	if err := b.Process.Signal(os.Interrupt); err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			return
		}
		b.t.Fatalf("could not stop process with PID: %d, err: %s", b.Process.Pid, err)
	}
}

// LogMatch tests each line of the logfile to see if contains any
// match of the provided regular expression. It will open the log
// file on every call, read until EOF, then close it. LogContains
// will be faster so use that if possible.
func (b *BeatProc) LogMatch(match string) bool {
	re := regexp.MustCompile(match)
	logFile := b.openLogFile()
	defer logFile.Close()

	found := false
	found, b.logFileOffset = b.logRegExpMatch(re, logFile, b.logFileOffset)
	if found {
		return found
	}

	eventLogFile := b.openEventLogFile()
	if eventLogFile == nil {
		return false
	}
	defer eventLogFile.Close()
	found, b.eventLogFileOffset = b.logRegExpMatch(re, eventLogFile, b.eventLogFileOffset)

	return found
}

func (b *BeatProc) logRegExpMatch(re *regexp.Regexp, logFile *os.File, offset int64) (bool, int64) {
	_, err := logFile.Seek(offset, io.SeekStart)
	if err != nil {
		b.t.Fatalf("could not set offset for '%s': %s", logFile.Name(), err)
	}

	defer func() {
		if err := logFile.Close(); err != nil {
			// That's not quite a test error, but it can impact
			// next executions of LogContains, so treat it as an error
			b.t.Errorf("could not close log file: %s", err)
		}
	}()

	r := bufio.NewReader(logFile)
	for {
		data, err := r.ReadBytes('\n')
		line := string(data)
		offset += int64(len(data))

		if err != nil {
			if err != io.EOF {
				b.t.Fatalf("error reading log file '%s': %s", logFile.Name(), err)
			}
			break
		}

		if re.MatchString(line) {
			return true, offset
		}
	}

	return false, offset
}

// LogContains looks for `s` as a substring of every log line,
// it will open the log file on every call, read it until EOF,
// then close it. It keeps track of the offset so subsequent calls
// will only read log entries that were not read by the previous
// call.
//
// The events log file is read after the normal log file and its
// offset is tracked separately.
func (b *BeatProc) LogContains(s string) bool {
	logFile := b.openLogFile()
	defer logFile.Close()

	found := false
	found, b.logFileOffset = b.searchStrInLogs(logFile, s, b.logFileOffset)
	if found {
		return found
	}

	eventLogFile := b.openEventLogFile()
	if eventLogFile == nil {
		return false
	}
	defer eventLogFile.Close()
	found, b.eventLogFileOffset = b.searchStrInLogs(eventLogFile, s, b.eventLogFileOffset)

	return found
}

// searchStrInLogs search for s as a substring of any line in logFile starting
// from offset.
//
// It will close logFile and return the current offset.
func (b *BeatProc) searchStrInLogs(logFile *os.File, s string, offset int64) (bool, int64) {
	t := b.t

	_, err := logFile.Seek(offset, io.SeekStart)
	if err != nil {
		t.Fatalf("could not set offset for '%s': %s", logFile.Name(), err)
	}

	defer func() {
		if err := logFile.Close(); err != nil {
			// That's not quite a test error, but it can impact
			// next executions of LogContains, so treat it as an error
			t.Errorf("could not close log file: %s", err)
		}
	}()

	r := bufio.NewReader(logFile)
	for {
		data, err := r.ReadBytes('\n')
		line := string(data)
		offset += int64(len(data))

		if err != nil {
			if err != io.EOF {
				t.Fatalf("error reading log file '%s': %s", logFile.Name(), err)
			}
			break
		}

		if strings.Contains(line, s) {
			return true, offset
		}
	}

	return false, offset
}

// WaitForLogs waits for the specified string s to be present in the logs within
// the given timeout duration and fails the test if s is not found.
// msgAndArgs should be a format string and arguments that will be printed
// if the logs are not found, providing additional context for debugging.
func (b *BeatProc) WaitForLogs(s string, timeout time.Duration, msgAndArgs ...any) {
	b.t.Helper()
	require.Eventually(b.t, func() bool {
		return b.LogContains(s)
	}, timeout, 100*time.Millisecond, msgAndArgs...)
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
	b.baseArgs = append(b.baseArgs, "-c", b.configFile)
}

// openGlobFile opens a file defined by glob. The glob must resolve to a single
// file otherwise the test fails. It returns a *os.File and a boolean indicating
// whether a file was found.
//
// If `waitForFile` is true, it will wait up to 5 seconds for the file to
// be created. The test will fail if the file is not found. If it is false
// and no file is found, nil and false are returned.
func (b *BeatProc) openGlobFile(glob string, waitForFile bool) *os.File {
	t := b.t

	files, err := filepath.Glob(glob)
	if err != nil {
		t.Fatalf("could not expand log file glob: %s", err)
	}

	if waitForFile && len(files) == 0 {
		require.Eventually(t, func() bool {
			files, err = filepath.Glob(glob)
			if err != nil {
				t.Fatalf("could not expand log file glob: %s", err)
			}
			return len(files) == 1
		}, 5*time.Second, 100*time.Millisecond,
			"waiting for log file matching glob '%s' to be created", glob)
	}

	// We only reach this line if `waitForFile` is false, so we need
	// to check whether we found a file
	if len(files) == 0 {
		return nil
	}

	f, err := os.Open(files[0])
	if err != nil {
		t.Fatalf("could not open log file '%s': %s", files[0], err)
	}

	return f
}

// openLogFile opens the log file for reading and returns it.
// It's the caller's responsibility to close the file.
// If the log file is not found, the test fails. The returned
// value is never nil.
func (b *BeatProc) openLogFile() *os.File {
	// Beats can produce two different log files, to make sure we're
	// reading the normal one we add the year to the glob. The default
	// log file name looks like: filebeat-20240116.ndjson
	year := time.Now().Year()
	glob := fmt.Sprintf("%s-%d*.ndjson", filepath.Join(b.tempDir, b.beatName), year)

	return b.openGlobFile(glob, true)
}

// openEventLogFile opens the log file for reading and returns it.
// If the events log file does not exist, nil is returned
// It's the caller's responsibility to close the file.
func (b *BeatProc) openEventLogFile() *os.File {
	// Beats can produce two different log files, to make sure we're
	// reading the normal one we add the year to the glob. The default
	// log file name looks like: filebeat-20240116.ndjson
	year := time.Now().Year()
	glob := fmt.Sprintf("%s-events-data-%d*.ndjson", filepath.Join(b.tempDir, b.beatName), year)

	return b.openGlobFile(glob, false)
}

// createTempDir creates a temporary directory that will be
// removed after the tests passes.
//
// If the test fails, the temporary directory is not removed.
//
// If the tests are run with -v, the temporary directory will
// be logged.
func createTempDir(t *testing.T) string {
	rootDir, err := filepath.Abs("../../build/integration-tests")
	if err != nil {
		t.Fatalf("failed to determine absolute path for temp dir: %s", err)
	}
	err = os.MkdirAll(rootDir, 0o750)
	if err != nil {
		t.Fatalf("error making test dir: %s: %s", rootDir, err)
	}
	tempDir, err := os.MkdirTemp(rootDir, strings.ReplaceAll(t.Name(), "/", "-"))
	if err != nil {
		t.Fatalf("failed to make temp directory: %s", err)
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
	esURL := GetESURL(t, "http")

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(500*time.Second))
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, esURL.String(), nil)
	if err != nil {
		t.Fatalf("cannot create request to ensure ES is running: %s", err)
	}

	u := esURL.User.Username()
	p, _ := esURL.User.Password()
	req.SetBasicAuth(u, p)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// If you're reading this message, you probably forgot to start ES
		// run `mage docker:composeUp` from Filebeat's folder to start all
		// containers required for integration tests
		t.Fatalf("cannot execute HTTP request to ES: '%s', check to make sure ES is running (mage docker:composeUp)", err)
	}
	defer resp.Body.Close()
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

	metaBytes, err := io.ReadAll(metaFile)
	require.NoError(b.t, err, "error reading meta file")
	err = json.Unmarshal(metaBytes, &m)
	require.NoError(b.t, err, "error unmarshalling meta data")
	return m, nil
}

// RemoveAllCLIArgs removes all CLI arguments configured.
// It will also remove all configuration for home path and
// logs, there fore some methods, like the ones that read logs,
// might fail if Filebeat is not configured the way this framework
// expects.
//
// The only CLI argument kept is the `--systemTest` that is necessary
// to make the System Test Binary run Filebeat
func (b *BeatProc) RemoveAllCLIArgs() {
	b.baseArgs = []string{b.beatName, "--systemTest"}
}

func (b *BeatProc) Stdin() io.WriteCloser {
	return b.stdin
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
		default:
			t.Fatalf("could not determine port from env variable: ES_PORT=%s", esPort)
		}
	}

	user := os.Getenv("ES_USER")
	if user == "" {
		user = "admin"
	}

	pass := os.Getenv("ES_PASS")
	if pass == "" {
		pass = "testing"
	}

	esURL := url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%s", esHost, esPort),
		User:   url.UserPassword(user, pass),
	}
	return esURL
}

func GetKibana(t *testing.T) (url.URL, *url.Userinfo) {
	t.Helper()

	kibanaHost := os.Getenv("KIBANA_HOST")
	if kibanaHost == "" {
		kibanaHost = "localhost"
	}

	kibanaPort := os.Getenv("KIBANA_PORT")
	if kibanaPort == "" {
		kibanaPort = "5601"
	}

	kibanaURL := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%s", kibanaHost, kibanaPort),
	}
	kibanaUser := url.UserPassword("beats", "testing")
	return kibanaURL, kibanaUser
}

func HttpDo(t *testing.T, method string, targetURL url.URL) (statusCode int, body []byte, err error) {
	t.Helper()
	client := &http.Client{}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(30*time.Second))
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, targetURL.String(), nil)
	if err != nil {
		return 0, nil, fmt.Errorf("error making request, method: %s, url: %s, error: %w", method, targetURL.String(), err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("error doing request, method: %s, url: %s, error: %w", method, targetURL.String(), err)
	}
	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)

	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("error reading request, method: %s, url: %s, status code: %d", method, targetURL.String(), resp.StatusCode)
	}
	return resp.StatusCode, body, nil
}

func FormatDatastreamURL(t *testing.T, srcURL url.URL, dataStream string) (url.URL, error) {
	t.Helper()
	path, err := url.JoinPath("/_data_stream", dataStream)
	if err != nil {
		return url.URL{}, fmt.Errorf("error joining data_stream path: %w", err)
	}
	srcURL.Path = path
	return srcURL, nil
}

func FormatIndexTemplateURL(t *testing.T, srcURL url.URL, template string) (url.URL, error) {
	t.Helper()
	path, err := url.JoinPath("/_index_template", template)
	if err != nil {
		return url.URL{}, fmt.Errorf("error joining index_template path: %w", err)
	}
	srcURL.Path = path
	return srcURL, nil
}

func FormatPolicyURL(t *testing.T, srcURL url.URL, policy string) (url.URL, error) {
	t.Helper()
	path, err := url.JoinPath("/_ilm/policy", policy)
	if err != nil {
		return url.URL{}, fmt.Errorf("error joining ilm policy path: %w", err)
	}
	srcURL.Path = path
	return srcURL, nil
}

func FormatRefreshURL(t *testing.T, srcURL url.URL) url.URL {
	t.Helper()
	srcURL.Path = "/_refresh"
	return srcURL
}

func FormatDataStreamSearchURL(t *testing.T, srcURL url.URL, dataStream string) (url.URL, error) {
	t.Helper()
	path, err := url.JoinPath("/", dataStream, "_search")
	if err != nil {
		return url.URL{}, fmt.Errorf("error joining ilm policy path: %w", err)
	}
	srcURL.Path = path
	return srcURL, nil
}

func readLastNBytes(filename string, numBytes int64) ([]byte, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening %s: %w", filename, err)
	}
	fInfo, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("error stating %s: %w", filename, err)
	}
	var startPosition int64
	if fInfo.Size() >= numBytes {
		startPosition = fInfo.Size() - numBytes
	} else {
		startPosition = 0
	}
	_, err = f.Seek(startPosition, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("error seeking to %d in %s: %w", startPosition, filename, err)
	}
	return io.ReadAll(f)
}

func reportErrors(t *testing.T, tempDir string, beatName string) {
	var maxlen int64 = 2048
	stderr, err := readLastNBytes(filepath.Join(tempDir, "stderr"), maxlen)
	if err != nil {
		t.Logf("error reading stderr: %s", err)
	}
	t.Logf("Last %d bytes of stderr:\n%s", len(stderr), string(stderr))

	stdout, err := readLastNBytes(filepath.Join(tempDir, "stdout"), maxlen)
	if err != nil {
		t.Logf("error reading stdout: %s", err)
	}
	t.Logf("Last %d bytes of stdout:\n%s", len(stdout), string(stdout))

	glob := fmt.Sprintf("%s-*.ndjson", filepath.Join(tempDir, beatName))
	files, err := filepath.Glob(glob)
	if err != nil {
		t.Logf("glob error with: %s: %s", glob, err)
	}
	for _, f := range files {
		contents, err := readLastNBytes(f, maxlen)
		if err != nil {
			t.Logf("error reading %s: %s", f, err)
		}
		t.Logf("Last %d bytes of %s:\n%s", len(contents), f, string(contents))
	}
}

// GenerateLogFile writes count lines to path, each line is 50 bytes.
// Each line contains the current time (RFC3339) and a counter
func GenerateLogFile(t *testing.T, path string, count int, append bool) {
	var file *os.File
	var err error
	if !append {
		file, err = os.Create(path)
		if err != nil {
			t.Fatalf("could not create file '%s': %s", path, err)
		}
	} else {
		file, err = os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
		if err != nil {
			t.Fatalf("could not open or create file: '%s': %s", path, err)
		}
	}

	defer func() {
		if err := file.Close(); err != nil {
			t.Fatalf("could not close file: %s", err)
		}
	}()
	defer func() {
		if err := file.Sync(); err != nil {
			t.Fatalf("could not sync file: %s", err)
		}
	}()
	now := time.Now().Format(time.RFC3339)
	// If the length is different, e.g when there is no offset from UTC.
	// add some padding so the length is predictable
	if len(now) != len(time.RFC3339) {
		paddingNeeded := len(time.RFC3339) - len(now)
		for i := 0; i < paddingNeeded; i++ {
			now += "-"
		}
	}
	for i := 0; i < count; i++ {
		if _, err := fmt.Fprintf(file, "%s           %13d\n", now, i); err != nil {
			t.Fatalf("could not write line %d to file: %s", count+1, err)
		}
	}
}
