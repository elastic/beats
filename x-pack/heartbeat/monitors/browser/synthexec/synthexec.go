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

	"github.com/elastic/beats/v7/heartbeat/beater"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const debugSelector = "synthexec"

func init() {
	beater.RegisterJourneyLister(ListJourneys)
}

// ListJourneys takes the given suite performs a dry run, capturing the Journey names, and returns the list.
func ListJourneys(ctx context.Context, suiteFile string, params common.MapStr) (journeyNames []string, err error) {
	dir, err := getSuiteDir(suiteFile)
	if err != nil {
		return nil, err
	}

	if os.Getenv("ELASTIC_SYNTHETICS_OFFLINE") != "true" {
		// Ensure all deps installed
		err = runSimpleCommand(exec.Command("npm", "install"), dir)
		if err != nil {
			return nil, err
		}

		// Update playwright, needs to run separately to ensure post-install hook is run that downloads
		// chrome. See https://github.com/microsoft/playwright/issues/3712
		err = runSimpleCommand(exec.Command("npm", "install", "playwright-chromium"), dir)
		if err != nil {
			return nil, err
		}
	}

	cmdFactory, err := suiteCommandFactory(dir, suiteFile, "--dry-run")
	if err != nil {
		return nil, err
	}

	mpx, err := runCmd(ctx, cmdFactory(), nil, params)
Outer:
	for {
		select {
		case se := <-mpx.SynthEvents():
			if se == nil {
				break Outer
			}
			if se.Type == "journey/register" {
				journeyNames = append(journeyNames, se.Journey.Name)
			}
		}
	}

	logp.Info("Discovered journeys %#v", journeyNames)
	return journeyNames, nil
}

// SuiteJob will run a single journey by name from the given suite.
func SuiteJob(ctx context.Context, suiteFile string, journeyName string, params common.MapStr) (jobs.Job, error) {
	newCmd, err := suiteCommandFactory(suiteFile, suiteFile, "--screenshots", "--journey-name", journeyName)
	if err != nil {
		return nil, err
	}

	return startCmdJob(ctx, newCmd, nil, params), nil
}

func suiteCommandFactory(suiteFile string, args ...string) (func() *exec.Cmd, error) {
	npmRoot, err := getNpmRoot(suiteFile)
	if err != nil {
		return nil, err
	}

	newCmd := func() *exec.Cmd {
		bin := filepath.Join(npmRoot, "node_modules/.bin/elastic-synthetics")
		cmd := exec.Command(bin, args...)
		cmd.Dir = npmRoot
		return cmd
	}

	return newCmd, nil
}

// InlineJourneyJob returns a job that runs the given source as a single journey.
func InlineJourneyJob(ctx context.Context, script string, params common.MapStr) jobs.Job {
	newCmd := func() *exec.Cmd {
		return exec.Command("elastic-synthetics", "--inline", "--screenshots")
	}

	return startCmdJob(ctx, newCmd, &script, params)
}

// startCmdJob adapts commands into a heartbeat job. This is a little awkward given that the command's output is
// available via a sequence of events in the multiplexer, while heartbeat jobs are tail recursive continuations.
// Here, we adapt one to the other, where each recursive job pulls another item off the chan until none are left.
func startCmdJob(ctx context.Context, newCmd func() *exec.Cmd, stdinStr *string, params common.MapStr) jobs.Job {
	return func(event *beat.Event) ([]jobs.Job, error) {
		mpx, err := runCmd(ctx, newCmd(), stdinStr, params)
		if err != nil {
			return nil, err
		}
		return []jobs.Job{readResultsJob(ctx, mpx.SynthEvents(), newJourneyEnricher())}, nil
	}
}

// readResultsJob adapts the output of an ExecMultiplexer into a Job, that uses continuations
// to read all output.
func readResultsJob(ctx context.Context, synthEvents <-chan *SynthEvent, je *journeyEnricher) jobs.Job {
	return func(event *beat.Event) (conts []jobs.Job, err error) {
		select {
		case se := <-synthEvents:
			err = je.enrich(event, se)
			if se != nil {
				return []jobs.Job{readResultsJob(ctx, synthEvents, je)}, err
			} else {
				return nil, err
			}
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
) (mpx *ExecMultiplexer, err error) {
	mpx = NewExecMultiplexer()
	// Setup a pipe for JSON structured output
	jsonReader, jsonWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	// Common args
	cmd.Env = append(os.Environ(), "NODE_ENV=production")
	// We need to pass both files in here otherwise we get a broken pipe, even
	// though node only touches the writer
	cmd.ExtraFiles = []*os.File{jsonWriter, jsonReader}
	cmd.Args = append(cmd.Args,
		// Out fd is always 3 since it's the only FD passed into cmd.ExtraFiles
		// see the docs for ExtraFiles in https://golang.org/pkg/os/exec/#Cmd
		"--json",
		"--network",
		"--outfd", "3",
	)
	if len(params) > 0 {
		paramsBytes, _ := json.Marshal(params)
		cmd.Args = append(cmd.Args, "--suite-params", string(paramsBytes))
	}

	logp.Info("Running command: %s in directory: '%s'", cmd.String(), cmd.Dir)

	if stdinStr != nil {
		logp.Debug(debugSelector, "Using stdin str %s", *stdinStr)
		cmd.Stdin = strings.NewReader(*stdinStr)
	}

	wg := sync.WaitGroup{}

	// Send stdout into the output
	stdoutPipe, err := cmd.StdoutPipe()
	wg.Add(1)
	go func() {
		scanToSynthEvents(stdoutPipe, stdoutToSynthEvent, mpx.writeSynthEvent)
		wg.Done()
	}()

	stderrPipe, err := cmd.StderrPipe()
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

	// Kill the process if the context ends
	go func() {
		select {
		case <-ctx.Done():
			cmd.Process.Kill()
		}
	}()

	// Close mpx after the process is done and all events have been sent / consumed
	go func() {
		err := cmd.Wait()
		if err != nil {
			logp.Err("Error waiting for command %s: %s", cmd.String(), err)
		}
		jsonWriter.Close()
		jsonReader.Close()
		logp.Info("Command has completed(%d): %s", cmd.ProcessState.ExitCode(), cmd.String())
		if err != nil {
			str := fmt.Sprintf("command exited with status %d: %s", cmd.ProcessState.ExitCode(), err)
			mpx.writeSynthEvent(&SynthEvent{
				Type:  "cmd/status",
				Error: &SynthError{Name: "cmdexit", Message: str},
			})
			logp.Warn("Error executing command '%s': %s", cmd.String(), err)
		}
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
		if scanner.Err() != nil {
			logp.Warn("Error scanning results %s", scanner.Err())
			return scanner.Err()
		}

		se, err := transform(scanner.Bytes(), scanner.Text())
		if err != nil {
			logp.Warn("error parsing line: %s for line: %s", err, scanner.Text())
			continue
		}
		if se != nil {
			cb(se)
		}
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
			TimestampEpochMicros: float64(time.Now().UnixNano() / int64(time.Millisecond)),
			Payload: map[string]interface{}{
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
		return nil, fmt.Errorf("Unmarshal succeeded, but no type found for: %s", text)
	}
	return
}

func getSuiteDir(suiteFile string) (string, error) {
	path, err := filepath.Abs(suiteFile)
	if err != nil {
		return "", err
	}
	stat, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if stat.IsDir() {
		return suiteFile, nil
	}

	return filepath.Dir(suiteFile), nil
}

func runSimpleCommand(cmd *exec.Cmd, dir string) error {
	cmd.Dir = dir
	logp.Info("Running %s in %s", cmd, dir)
	output, err := cmd.CombinedOutput()
	logp.Info("Ran %s got %s", cmd, string(output))
	return err
}

func getNpmRoot(path string) (string, error) {
	candidate := filepath.Join(path, "package.json")
	_, err := os.Lstat(candidate)
	if err == nil {
		return path, nil
	}
	// Try again one level up
	parent := filepath.Dir(path)
	if len(parent) < 2 {
		return "", fmt.Errorf("no package.json found")
	}
	return getNpmRoot(parent)
}
