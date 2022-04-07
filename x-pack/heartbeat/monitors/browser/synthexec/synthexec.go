// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package synthexec

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const debugSelector = "synthexec"

type StdSuiteFields struct {
	Name     string
	Id       string
	Type     string
	IsInline bool
}

type FilterJourneyConfig struct {
	Tags  []string `config:"tags"`
	Match string   `config:"match"`
}

// SuiteJob will run a single journey by name from the given suite.
func SuiteJob(ctx context.Context, suitePath string, params common.MapStr, filterJourneys FilterJourneyConfig, fields StdSuiteFields, extraArgs ...string) (jobs.Job, error) {
	// Run the command in the given suitePath, use '.' as the first arg since the command runs
	// in the correct dir
	cmdFactory, err := suiteCommandFactory(suitePath, extraArgs...)
	if err != nil {
		return nil, err
	}

	return startCmdJob(ctx, cmdFactory, nil, params, filterJourneys, fields), nil
}

func suiteCommandFactory(suitePath string, args ...string) (func() *exec.Cmd, error) {
	npmRoot, err := getNpmRoot(suitePath)
	if err != nil {
		return nil, err
	}

	newCmd := func() *exec.Cmd {
		bin := filepath.Join(npmRoot, "node_modules/.bin/elastic-synthetics")
		// Always put the suite path first to prevent conflation with variadic args!
		// See https://github.com/tj/commander.js/blob/master/docs/options-taking-varying-arguments.md
		// Note, we don't use the -- approach because it's cleaner to always know we can add new options
		// to the end.
		cmd := exec.Command(bin, append([]string{suitePath}, args...)...)
		cmd.Dir = npmRoot
		return cmd
	}

	return newCmd, nil
}

// InlineJourneyJob returns a job that runs the given source as a single journey.
func InlineJourneyJob(ctx context.Context, script string, params common.MapStr, fields StdSuiteFields, extraArgs ...string) jobs.Job {
	newCmd := func() *exec.Cmd {
		return exec.Command("elastic-synthetics", append(extraArgs, "--inline")...)
	}

	return startCmdJob(ctx, newCmd, &script, params, FilterJourneyConfig{}, fields)
}

// startCmdJob adapts commands into a heartbeat job. This is a little awkward given that the command's output is
// available via a sequence of events in the multiplexer, while heartbeat jobs are tail recursive continuations.
// Here, we adapt one to the other, where each recursive job pulls another item off the chan until none are left.
func startCmdJob(ctx context.Context, newCmd func() *exec.Cmd, stdinStr *string, params common.MapStr, filterJourneys FilterJourneyConfig, fields StdSuiteFields) jobs.Job {
	return func(event *beat.Event) ([]jobs.Job, error) {
		mpx, err := runCmd(ctx, newCmd(), stdinStr, params, filterJourneys)
		if err != nil {
			return nil, err
		}
		senr := streamEnricher{}
		return []jobs.Job{readResultsJob(ctx, mpx.SynthEvents(), senr.enrich, fields)}, nil
	}
}

// readResultsJob adapts the output of an ExecMultiplexer into a Job, that uses continuations
// to read all output.
func readResultsJob(ctx context.Context, synthEvents <-chan *SynthEvent, enrich enricher, fields StdSuiteFields) jobs.Job {
	return func(event *beat.Event) (conts []jobs.Job, err error) {
		se := <-synthEvents
		err = enrich(event, se, fields)
		if se != nil {
			return []jobs.Job{readResultsJob(ctx, synthEvents, enrich, fields)}, err
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
	params common.MapStr,
	filterJourneys FilterJourneyConfig,
) (mpx *ExecMultiplexer, err error) {
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
	loggableCmd := exec.Command(cmd.Path, cmd.Args...)
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
		scanToSynthEvents(stdoutPipe, stdoutToSynthEvent, mpx.writeSynthEvent)
		wg.Done()
	}()

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("could not open stderr pipe: %w", err)
	}
	wg.Add(1)
	go func() {
		scanToSynthEvents(stderrPipe, stderrToSynthEvent, mpx.writeSynthEvent)
		wg.Done()
	}()

	// Send the test results into the output
	wg.Add(1)
	go func() {
		scanToSynthEvents(jsonReader, jsonToSynthEvent, mpx.writeSynthEvent)
		wg.Done()
	}()
	err = cmd.Start()
	if err != nil {
		logp.Warn("Could not start command %s: %s", cmd, err)
		return nil, err
	}

	// Kill the process if the context ends
	go func() {
		<-ctx.Done()
		cmd.Process.Kill()
	}()

	// Close mpx after the process is done and all events have been sent / consumed
	go func() {
		err := cmd.Wait()
		jsonWriter.Close()
		jsonReader.Close()
		logp.Info("Command has completed(%d): %s", cmd.ProcessState.ExitCode(), loggableCmd.String())

		var cmdError *SynthError = nil
		if err != nil {
			errMessage := fmt.Sprintf("command exited with status %d: %s", cmd.ProcessState.ExitCode(), err)
			cmdError = &SynthError{Name: "cmdexit", Message: errMessage}
			logp.Warn("Error executing command '%s' (%d): %s", loggableCmd.String(), cmd.ProcessState.ExitCode(), err)
		}

		mpx.writeSynthEvent(&SynthEvent{
			Type:                 "cmd/status",
			Error:                cmdError,
			TimestampEpochMicros: float64(time.Now().UnixMicro()),
		})

		wg.Wait()
		mpx.Close()
	}()

	return mpx, nil
}

// scanToSynthEvents takes a reader, a transform function, and a callback, and processes
// each scanned line via the reader before invoking it with the callback.
func scanToSynthEvents(rdr io.ReadCloser, transform func(bytes []byte, text string) (*SynthEvent, error), cb func(*SynthEvent)) error {
	scanner := bufio.NewScanner(rdr)
	buf := make([]byte, 1024*1024*2)  // 2MiB initial buffer (images can be big!)
	scanner.Buffer(buf, 1024*1024*40) // Max 50MiB Buffer

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

var stdoutToSynthEvent = lineToSynthEventFactory("stdout")
var stderrToSynthEvent = lineToSynthEventFactory("stderr")

// lineToSynthEventFactory is a factory that can take a line from the scanner and transform it into a *SynthEvent.
func lineToSynthEventFactory(typ string) func(bytes []byte, text string) (res *SynthEvent, err error) {
	return func(bytes []byte, text string) (res *SynthEvent, err error) {
		logp.Info("%s: %s", typ, text)
		return &SynthEvent{
			Type:                 typ,
			TimestampEpochMicros: float64(time.Now().UnixMicro()),
			Payload: common.MapStr{
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
	return
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
