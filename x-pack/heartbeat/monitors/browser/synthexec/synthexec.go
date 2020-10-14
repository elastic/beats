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

package synthexec

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/beats/v7/heartbeat/beater"
	"github.com/elastic/beats/v7/libbeat/beat"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	//"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const debugSelector = "synthexec"

func init() {
	beater.RegisterJourneyLister(ListJourneys)
}

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
			if se.Type == "journey/register" {
				journeyNames = append(journeyNames, se.Journey.Name)
			}
		}
	}

	logp.Info("Discovered journeys %#v", journeyNames)
	return journeyNames, nil
}

// SuiteJob will run a single journey by name from the given suite file.
func SuiteJob(ctx context.Context, suiteFile string, journeyName string, params common.MapStr) jobs.Job {
	newCmd := func() *exec.Cmd {
		return exec.Command(
			"node",
			suiteFile,
			"--screenshots",
			"--journey-name", journeyName,
		)
	}

	return startCmdJob(ctx, newCmd, nil, params)
}

// JourneyJob returns a job that runs the given source as a single journey.
func JourneyJob(ctx context.Context, script string, params common.MapStr) jobs.Job {
	newCmd := func() *exec.Cmd {
		return exec.Command(
			"npx",
			"@elastic/synthetics",
			"--inline",
			"--screenshots",
		)
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
		return []jobs.Job{readResultsJob(ctx, mpx, readResultsState{})}, nil
	}

}

type readResultsState struct {
	journeyComplete bool
	errorCount int
	lastError error
	stepCount int
}

// readResultsJob creates adapts the output of an ExecMultiplexer into a Job, that uses continuations
// to read all output.
func readResultsJob(ctx context.Context, mpx *ExecMultiplexer, state readResultsState) jobs.Job {
	return func(event *beat.Event) (conts []jobs.Job, err error) {
		select {
		case se := <-mpx.SynthEvents():
			// No more events? In this case this is the summary event
			if se == nil {
				if state.journeyComplete {
					return nil, state.lastError
				}
				return nil, fmt.Errorf("journey did not finish executing, %d steps ran", state.stepCount)
			}
			if se.TimestampEpochMillis != 0 {
				event.Timestamp = time.Unix(int64(se.TimestampEpochMillis/1000), (int64(se.TimestampEpochMillis) % 1000)*1000000)
			}
			switch se.Type {
			case "journey/end":
				state.journeyComplete = true
			case "step/end":
				state.stepCount++
			}

			eventext.MergeEventFields(event, se.ToMap())
			var jobErr error
			if se.Error != nil {
				jobErr = fmt.Errorf("error executing step: %s", se.Error.String())
				state.errorCount++
				state.lastError = jobErr
			}
			return []jobs.Job{readResultsJob(ctx, mpx, state)}, jobErr
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
			TimestampEpochMillis: float64(time.Now().UnixNano() / int64(time.Millisecond)),
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
