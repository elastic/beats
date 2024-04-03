// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin || synthetics

package synthexec

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/heartbeat/ecserr"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/libbeat/beat"
)

const debugSelector = "synthexec"

type FilterJourneyConfig struct {
	Tags  []string `config:"tags"`
	Match string   `config:"match"`
}

// platformCmdMutate is the hook for OS specific mutation of cmds
// This is practically just used by synthexec_unix.go to add Sysprocattrs
// It's still nice for devs to be able to test browser monitors on OSX
// where these are unsupported
var platformCmdMutate func(*exec.Cmd) = func(*exec.Cmd) {}

var SynthexecTimeout struct{}

// ProjectJob will run a single journey by name from the given project.
func ProjectJob(ctx context.Context, projectPath string, params mapstr.M, filterJourneys FilterJourneyConfig, fields stdfields.StdMonitorFields, extraArgs ...string) (jobs.Job, error) {
	// Run the command in the given projectPath, use '.' as the first arg since the command runs
	// in the correct dir
	cmdFactory, err := projectCommandFactory(projectPath, extraArgs...)
	if err != nil {
		return nil, err
	}

	return startCmdJob(ctx, cmdFactory, nil, params, filterJourneys, fields), nil
}

func projectCommandFactory(projectPath string, args ...string) (func() *exec.Cmd, error) {
	npmRoot, err := getNpmRoot(projectPath)
	if err != nil {
		return nil, err
	}

	newCmd := func() *exec.Cmd {
		bin := filepath.Join(npmRoot, "node_modules/.bin/elastic-synthetics")
		// Always put the project path first to prevent conflation with variadic args!
		// See https://github.com/tj/commander.js/blob/master/docs/options-taking-varying-arguments.md
		// Note, we don't use the -- approach because it's cleaner to always know we can add new options
		// to the end.
		cmd := exec.Command(bin, append([]string{projectPath}, args...)...)
		cmd.Dir = npmRoot
		return cmd
	}

	return newCmd, nil
}

// InlineJourneyJob returns a job that runs the given source as a single journey.
func InlineJourneyJob(ctx context.Context, script string, params mapstr.M, fields stdfields.StdMonitorFields, extraArgs ...string) jobs.Job {
	newCmd := func() *exec.Cmd {
		return exec.Command("elastic-synthetics", append(extraArgs, "--inline")...) //nolint:gosec // we are safely building a command here, users can add args at their own risk
	}

	return startCmdJob(ctx, newCmd, &script, params, FilterJourneyConfig{}, fields)
}

// startCmdJob adapts commands into a heartbeat job. This is a little awkward given that the command's output is
// available via a sequence of events in the multiplexer, while heartbeat jobs are tail recursive continuations.
// Here, we adapt one to the other, where each recursive job pulls another item off the chan until none are left.
func startCmdJob(ctx context.Context, newCmd func() *exec.Cmd, stdinStr *string, params mapstr.M, filterJourneys FilterJourneyConfig, sFields stdfields.StdMonitorFields) jobs.Job {
	return func(event *beat.Event) ([]jobs.Job, error) {
		senr := newStreamEnricher(sFields)
		mpx, err := runCmd(ctx, newCmd(), stdinStr, params, filterJourneys)
		if err != nil {
			err := senr.enrich(event, &SynthEvent{
				Type:  "cmd/could_not_start",
				Error: ECSErrToSynthError(ecserr.NewSyntheticsCmdCouldNotStartErr(err)),
			})
			return nil, err
		}
		// We don't just return the readResultsJob, otherwise we'd just send an empty event, execute it right away
		// then it'll keep executing itself until we're truly done
		return readResultsJob(ctx, mpx.SynthEvents(), senr.enrich)(event)
	}
}

// readResultsJob adapts the output of an ExecMultiplexer into a Job, that uses continuations
// to read all output.
func readResultsJob(ctx context.Context, synthEvents <-chan *SynthEvent, enrich enricher) jobs.Job {
	return func(event *beat.Event) (conts []jobs.Job, err error) {
		se := <-synthEvents
		err = enrich(event, se)
		if se != nil {
			return []jobs.Job{readResultsJob(ctx, synthEvents, enrich)}, err
		} else {
			return nil, err
		}
	}
}

// runCmd runs the given command, piping stdinStr if present to the command's stdin, and supplying
// the params var as a CLI argument.
func runCmd(
	ctx context.Context,
	cmd *exec.Cmd,
	stdinStr *string,
	params mapstr.M,
	filterJourneys FilterJourneyConfig,
) (mpx *ExecMultiplexer, err error) {
	// Attach sysproc attrs to ensure subprocesses are properly killed
	platformCmdMutate(cmd)

	mpx = NewExecMultiplexer()
	// Setup a pipe for JSON structured output
	jsonReader, jsonWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	// Common args
	cmd.Env = append(os.Environ(), "NODE_ENV=production")
	cmd.Args = append(cmd.Args, "--rich-events")

	if len(filterJourneys.Tags) > 0 {
		cmd.Args = append(cmd.Args, "--tags", strings.Join(filterJourneys.Tags, " "))
	}

	if filterJourneys.Match != "" {
		cmd.Args = append(cmd.Args, "--match", filterJourneys.Match)
	}

	// Variant of the command with no params, which could contain sensitive stuff
	loggableCmd := exec.Command(cmd.Path, cmd.Args...) //nolint:gosec // we are safely building a command here...
	if len(params) > 0 {
		paramsBytes, _ := json.Marshal(params)
		cmd.Args = append(cmd.Args, "--params", string(paramsBytes))
		loggableCmd.Args = append(loggableCmd.Args, "--params", fmt.Sprintf("\"{%d hidden params}\"", len(params)))
	}

	// We need to pass both files in here otherwise we get a broken pipe, even
	// though node only touches the writer
	cmd.ExtraFiles = []*os.File{jsonWriter, jsonReader}
	// Out fd is always 3 since it's the only FD passed into cmd.ExtraFiles
	// see the docs for ExtraFiles in https://golang.org/pkg/os/exec/#Cmd
	cmd.Args = append(cmd.Args, "--outfd", "3")

	logp.Info("Running command: %s in directory: '%s'", loggableCmd.String(), cmd.Dir)

	if stdinStr != nil {
		logp.Debug(debugSelector, "Using stdin str %s", *stdinStr)
		cmd.Stdin = strings.NewReader(*stdinStr)
	}

	wg := sync.WaitGroup{}

	// Send stdout into the output
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("could not open stdout pipe: %w", err)
	}
	wg.Add(1)
	go func() {
		err := scanToSynthEvents(stdoutPipe, stdoutToSynthEvent, mpx.writeSynthEvent)
		if err != nil {
			logp.Warn("could not scan stdout events from synthetics: %s", err)
		}

		wg.Done()
	}()

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("could not open stderr pipe: %w", err)
	}
	wg.Add(1)
	go func() {
		err := scanToSynthEvents(stderrPipe, stderrToSynthEvent, mpx.writeSynthEvent)
		if err != nil {
			logp.Warn("could not scan stderr events from synthetics: %s", err)
		}
		wg.Done()
	}()

	// Send the test results into the output
	wg.Add(1)
	go func() {
		defer jsonReader.Close()

		// We don't use scanToSynthEvents here because all lines here will be JSON
		// It's more efficient to let the json decoder handle the ndjson than
		// using the scanner
		decoder := json.NewDecoder(jsonReader)
		for {
			var se SynthEvent
			err := decoder.Decode(&se)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				logp.L().Warnf("error decoding json for test json results: %w", err)
			}

			mpx.writeSynthEvent(&se)
		}

		wg.Done()
	}()

	// This use of channels for results is awkward, but required for the thread locking below
	cmdStarted := make(chan error)
	cmdDone := make(chan error)
	go func() {
		// We must idle this thread and ensure it is not killed while the external program is running
		// see https://github.com/golang/go/issues/27505#issuecomment-713706104 . Otherwise, the Pdeathsig
		// could cause the subprocess to die prematurely
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		err = cmd.Start()

		cmdStarted <- err

		err := cmd.Wait()
		cmdDone <- err
	}()

	err = <-cmdStarted
	if err != nil {
		logp.Warn("Could not start command %s: %s", cmd, err)
		return nil, err
	}

	// Get timeout from parent ctx
	timeout, _ := ctx.Value(SynthexecTimeout).(time.Duration)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	go func() {
		<-ctx.Done()

		// ProcessState can be null if it hasn't reported back yet
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			return
		}

		err := cmd.Process.Kill()
		if err != nil {
			logp.Warn("could not kill synthetics process: %s", err)
		}
	}()

	// Close mpx after the process is done and all events have been sent / consumed
	go func() {
		err := <-cmdDone
		_ = jsonWriter.Close()
		logp.Info("Command has completed(%d): %s", cmd.ProcessState.ExitCode(), loggableCmd.String())

		var cmdError *SynthError = nil
		if err != nil {
			// err could be generic or it could have been killed by context timeout, log and check context
			// to decide which error to stream
			logp.Warn("Error executing command '%s' (%d): %s", loggableCmd.String(), cmd.ProcessState.ExitCode(), err)

			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				timeout, _ := ctx.Value(SynthexecTimeout).(time.Duration)
				cmdError = ECSErrToSynthError(ecserr.NewCmdTimeoutStatusErr(timeout, loggableCmd.String()))
			} else {
				cmdError = ECSErrToSynthError(ecserr.NewBadCmdStatusErr(cmd.ProcessState.ExitCode(), loggableCmd.String()))
			}
		}

		mpx.writeSynthEvent(&SynthEvent{
			Type:                 CmdStatus,
			Error:                cmdError,
			TimestampEpochMicros: float64(time.Now().UnixMicro()),
		})

		wg.Wait()
		mpx.Close()
		cancel()
	}()

	return mpx, nil
}

// scanToSynthEvents takes a reader, a transform function, and a callback, and processes
// each scanned line via the reader before invoking it with the callback.
func scanToSynthEvents(rdr io.ReadCloser, transform func(bytes []byte, text string) (*SynthEvent, error), cb func(*SynthEvent)) error {
	defer func() {
		_ = rdr.Close()
	}()
	scanner := bufio.NewScanner(rdr)
	buf := make([]byte, 1024*10)      // 10KiB initial buffer
	scanner.Buffer(buf, 1024*1024*10) // Max 10MiB Buffer

	for scanner.Scan() {
		se, err := transform(scanner.Bytes(), scanner.Text())
		if err != nil {
			logp.Warn("error parsing line: %s for line: %s", err, scanner.Text())
			continue
		}
		if se != nil {
			cb(se)
		}
	}

	if scanner.Err() != nil {
		logp.Warn("error scanning synthetics runner results %s", scanner.Err())
		return scanner.Err()
	}

	return nil
}

var stdoutToSynthEvent = lineToSynthEventFactory(Stdout)
var stderrToSynthEvent = lineToSynthEventFactory(Stderr)

// lineToSynthEventFactory is a factory that can take a line from the scanner and transform it into a *SynthEvent.
func lineToSynthEventFactory(typ string) func(bytes []byte, text string) (res *SynthEvent, err error) {
	return func(bytes []byte, text string) (res *SynthEvent, err error) {
		logp.Info("%s: %s", typ, text)
		return &SynthEvent{
			Type:                 typ,
			TimestampEpochMicros: float64(time.Now().UnixMicro()),
			Payload: mapstr.M{
				"message": text,
			},
		}, nil
	}
}

var emptyStringRegexp = regexp.MustCompile(`^\s*$`)

// jsonToSynthEvent can take a line from the scanner and transform it into a *SynthEvent. Will return
// nil res on empty lines.
func jsonToSynthEvent(bytes []byte, text string) (res *SynthEvent, err error) {
	// Skip empty lines
	if emptyStringRegexp.Match(bytes) {
		return nil, nil
	}

	res = &SynthEvent{}
	err = json.Unmarshal(bytes, res)

	if err != nil {
		return nil, err
	}

	if res.Type == "" {
		return nil, fmt.Errorf("unmarshal succeeded, but no type found for: %s", text)
	}
	return res, err
}

// getNpmRoot gets the closest ancestor path that contains package.json.
func getNpmRoot(path string) (string, error) {
	return getNpmRootIn(path, path)
}

// getNpmRootIn does the same as getNpmRoot but remembers the original path for
// debugging.
func getNpmRootIn(path, origPath string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("cannot check for package.json in empty path: '%s'", origPath)
	}
	candidate := filepath.Join(path, "package.json")
	_, err := os.Lstat(candidate)
	if err == nil {
		return path, nil
	}
	// Try again one level up
	parent := filepath.Dir(path)
	if len(parent) < 2 {
		return "", fmt.Errorf("no package.json found in '%s'", origPath)
	}
	return getNpmRootIn(parent, origPath)
}
