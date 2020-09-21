// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package synthexec

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

const debugSelector = "synthexec"

// ListJourneys takes the given suite perfors a dry run, capturing the Journey names, and returns the list.
func ListJourneys(ctx context.Context, suiteFile string, params common.MapStr) (journeyNames []string, err error) {
	cmd := exec.Command(
		"node",
		suiteFile,
		"--dry-run",
	)

	mpx, err := runCmd(ctx, cmd, nil, params)
Outer:
	for {
		select {
		case se := <-mpx.SynthEvents():
			if se == nil {
				break Outer
			}
			if se.Type == "journey/start" {
				journeyNames = append(journeyNames, se.Journey.Name)
			}
		}
	}

	logp.Info("Discovered journeys %#v", journeyNames)
	return journeyNames, nil
}

// SuiteJob will run a single journey by name from the given suite file.
func SuiteJob(ctx context.Context, suiteFile string, journeyName string, params common.MapStr, ) jobs.Job {
	cmd := exec.Command(
		"node",
		suiteFile,
		"--screenshots",
		"--journey-name", journeyName,
	)

	return cmdJob(ctx, cmd, nil, params)
}

// JourneyJob returns a job tha runs the given source as a single journey.
func JourneyJob(ctx context.Context, script string, params common.MapStr) jobs.Job {
	cmd := exec.Command(
		"npx",
		"@elastic/synthetics",
		"--stdin",
		"--screenshots",
	)

	return cmdJob(ctx, cmd, &script, params)
}

// cmdJob adapts commands into a heartbeat job. This is a little awkward given that the command's output is
// available via a sequence of events in the multiplexer, while heartbeat jobs are tail recursive continuations.
// Here, we adapt one to the other, where each recursive job pulls another item off the chan until none are left.
func cmdJob(ctx context.Context, cmd *exec.Cmd, stdinStr *string, params common.MapStr) jobs.Job {
	var j jobs.Job
	var mpx *ExecMultiplexer
	j = func(event *beat.Event) (conts []jobs.Job, err error) {
		if mpx == nil { // Start the script on the first invocation of this job
			mpx, err = runCmd(ctx, cmd, stdinStr, params)
		}
		select {
		case se := <-mpx.SynthEvents():
			if se == nil {
				return nil, nil
			}
			var emptyTime time.Time
			if se.Timestamp != emptyTime {
				event.Timestamp = se.Timestamp
			}

			eventext.MergeEventFields(event, se.ToMap())
			return []jobs.Job{j}, nil
		}
	}

	return j
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
		"--outfd", "3",
		"--json",
		"--network",
	)
	if len(params) > 0 {
		paramsBytes, _ := json.Marshal(params)
		cmd.Args = append(cmd.Args, "--suite-params", string(paramsBytes))
	}

	logp.Info("Running command: %s", cmd.String())

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
		jsonWriter.Close()
		jsonReader.Close()
		logp.Debug(debugSelector, "Command has completed %d", cmd.ProcessState.ExitCode())
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
func scanToSynthEvents(rdr io.ReadCloser, transform func(bytes []byte, text string) *SynthEvent, cb func(*SynthEvent)) error {
	scanner := bufio.NewScanner(rdr)
	buf := make([]byte, 1024*1024*2)   // 2MiB initial buffer
	scanner.Buffer(buf, 1024*1024*100) // Max 100MiB Buffer

	for scanner.Scan() {
		if scanner.Err() != nil {
			logp.Warn("Error scanning results %s", scanner.Err())
			return scanner.Err()
		}

		se := transform(scanner.Bytes(), scanner.Text())
		if se != nil {
			cb(se)
		}
	}

	return nil
}

var stdoutToSynthEvent = consoleLineToSynthEvent("stdout")
var stderrToSynthEvent = consoleLineToSynthEvent("stderr")

// consoleLineToSynthEvent is a factory that can take a line from the scanner and transform it into a *SynthEvent.
func consoleLineToSynthEvent(typ string) func(bytes []byte, text string) (res *SynthEvent) {
	return func(bytes []byte, text string) (res *SynthEvent) {
		logp.Info("%s: %s", typ, text)
		return &SynthEvent{
			Type:      typ,
			Timestamp: time.Now(),
			Payload: map[string]interface{}{
				"message": text,
			},
		}
	}
}

var emptyStringRegexp = regexp.MustCompile(`^\s*$`)

// jsonToSynthEvent can take a line from the scanner and transform it into a *SynthEvent.
func jsonToSynthEvent(bytes []byte, text string) (res *SynthEvent) {
	// Skip empty lines
	if emptyStringRegexp.Match(bytes) {
		return nil
	}

	err := json.Unmarshal(bytes, &res)
	if err != nil {
		logp.Warn("Could not unmarshal %s from synthexec: %s", text, err)
	}

	if res.Type == "" {
		logp.Warn("Unmarshal succeeded, but no type found for: %s", text)
		return nil
	}
	return
}
