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
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"io"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"
)

const debugSelector = "synthexec"

func ListJourneys(ctx context.Context, suiteFile string, params common.MapStr) (journeyNames []string, err error) {
	cmd := exec.Command(
		"node",
		suiteFile,
		"-e", "production",
		"--json",
		"--headless",
		"--dry-run",
	)

	mpx, err := runCmd(ctx, cmd, nil, params)
	running := true
	for running {
		select {
		case <- mpx.Done():
			running = false
		case se := <- mpx.SynthEvents():
			if se.Type == "" {
				running = false
			}
			if se.Type == "journey/start" {
				journeyNames = append(journeyNames, se.Journey.Name)
			}
		}
	}

	logp.Info("Discovered journeys %#v", journeyNames)
	return journeyNames, nil
}

func SuiteJob(ctx context.Context, suiteFile string, journeyName string, params common.MapStr, ) jobs.Job {
	cmd := exec.Command(
		"node",
		suiteFile,
		"--screenshots",
		"--journey-name", journeyName,
	)

	return cmdJob(ctx, cmd, nil, params)
}

func ScriptJob(ctx context.Context, script string, params common.MapStr) jobs.Job {
	cmd := exec.Command(
		"npx",
		"@elastic/synthetics",
		"--stdin",
		"--screenshots",
	)

	return cmdJob(ctx, cmd, &script, params)
}

func cmdJob(ctx context.Context, cmd *exec.Cmd, stdinStr *string, params common.MapStr) jobs.Job {
	var j jobs.Job
	var mpx *ExecMultiplexer
	j = func (event *beat.Event) (conts []jobs.Job, err error) {
		if mpx == nil { // Start the script on the first invocation of this job
			mpx, err = runCmd(ctx, cmd, stdinStr, params)
		}
		select {
		case se := <- mpx.SynthEvents():
			var emptyTime time.Time
			if se.Timestamp != emptyTime {
				event.Timestamp = se.Timestamp
			}

			eventext.MergeEventFields(event, se.ToMap())
			return []jobs.Job{j}, nil
		case <- mpx.Done():
			return nil, nil
		}
	}

	return j
}

func runCmd(
	ctx context.Context,
	cmd *exec.Cmd,
	stdinStr *string,
	params common.MapStr,
	) (mpx *ExecMultiplexer, err error) {
	mpx = NewExecMultiplexer()
	// Setup a pipe for structured output
	resultsReader, resultsWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	// Common args
	cmd.Env = append(os.Environ(), "NODE_ENV=production")
	// We need to pass both files in here otherwise we get a broken pipe, even
	// though node only touches the writer
	cmd.ExtraFiles = []*os.File{resultsWriter, resultsReader}
	cmd.Args = append(cmd.Args,
		// Out fd is always 3 since it's the only FD passed into cmd.ExtraFiles
		// see the docs for ExtraFiles in https://golang.org/pkg/os/exec/#Cmd
		"--outfd", "3",
		"--json",
		"--headless",
	)
	if len(params) > 0 {
		paramsBytes, _ := json.Marshal(params)
		cmd.Args = append(cmd.Args, "--suite-params", string(paramsBytes))
	}

	logp.Info("Running command: %s", cmd.String())

	if stdinStr	!= nil {
		logp.Debug(debugSelector, "Using stdin str %s", *stdinStr)
		cmd.Stdin = strings.NewReader(*stdinStr)
	}

	wg := sync.WaitGroup{}

	// Send stdout into the output
	stdoutPipe, err := cmd.StdoutPipe()
	wg.Add(1)
	go func() {
		sendConsoleLines(stdoutPipe, "console/stdout", mpx.writeSynthEvent)
		wg.Done()
	}()

	stderrPipe, err := cmd.StderrPipe()
	wg.Add(1)
	go func() {
		sendConsoleLines(stderrPipe, "console/stderr", mpx.writeSynthEvent)
		wg.Done()
	}()

	// Send the test results into the output
	wg.Add(1)
	go func() {
		sendSynthEvents(resultsReader, mpx.writeSynthEvent)
		wg.Done()
	}()
	err = cmd.Start()

	// Kill the process if the context ends
	go func() {
		select {
		case <- ctx.Done():
			cmd.Process.Kill()
		}
	}()

	// Close mpx after the process is done and all events have been sent / consumed
	go func() {
		err := cmd.Wait()
		resultsWriter.Close()
		resultsReader.Close()
		logp.Debug(debugSelector, "command has completed %d", cmd.ProcessState.ExitCode())
		if err != nil {
			str := fmt.Sprintf("command exited with status %d: %s", cmd.ProcessState.ExitCode(), err)
			mpx.writeSynthEvent(SynthEvent{
				Type: "cmd/status",
				Error: &str,
			})
			logp.Warn("Error executing command '%s': %s", cmd.String(), err)
		}
		wg.Wait()
		mpx.Close()
	}()

	return mpx, nil
}

func sendConsoleLines(rdr io.Reader, typ string, cb func(se SynthEvent)) {
	scanner := bufio.NewScanner(rdr)
	buf := make([]byte, 1024*1024*2) // 2MiB initial buffer
	scanner.Buffer(buf, 1024*1024*200) // Max 200MiB Buffer
	for scanner.Scan() {
		if scanner.Err() != nil {
			logp.Debug("could not scan console line: %s. Line was %s", scanner.Err(), scanner.Text())
		}
		logp.Info("%s: %s", typ, scanner.Text())
		if cb != nil {
			cb(SynthEvent{
				Type: typ,
				Timestamp: time.Now(),
				Payload: map[string]interface{}{
					"message": scanner.Text(),
				},
			})
		}
	}
}

func sendSynthEvents(rdr io.Reader, cb func(SynthEvent)) error {
	scanner := bufio.NewScanner(rdr)
	buf := make([]byte, 1024*1024*2) // 2MiB initial buffer
	scanner.Buffer(buf, 1024*1024*100) // Max 100MiB Buffer

	emptyString := regexp.MustCompile(`^\s*$`)

	for scanner.Scan() {
		if scanner.Err() != nil {
			logp.Warn("Error scanning results %s", scanner.Err())
			return scanner.Err()
		}

		// Skip empty lines
		if emptyString.Match(scanner.Bytes()) {
			continue
		}

		var res = SynthEvent{}
		err := json.Unmarshal(scanner.Bytes(), &res)
		if err != nil {
			logp.Warn("Could not unmarshal %s from synthexec: %s", scanner.Text(), err)
		}

		if res.Type == "" {
			logp.Warn("Unmarshal succeeded, but no type found for: %s", scanner.Text())
			continue
		}

		if err == nil && cb != nil {
			cb(res)
		}
	}
	return nil
}

type SynthEvent struct {
	Type           string                 `json:"type"`
	PackageVersion string                 `json:"package_version"`
	Index          int                    `json:"index""`
	Step           *Step                   `json:"step"`
	Journey        *Journey                `json:"journey"`
	Timestamp      time.Time              `json:"@timestamp"`
	Payload        map[string]interface{} `json:"payload"`
	Blob           *string                 `json:"blob"`
	Error          *string                 `json:"error"`
	URL            *string                 `json:"url"`
}

func (se SynthEvent) ToMap() common.MapStr {
	// We don't add @timestamp to the map string since that's specially handled in beat.Event
	e := common.MapStr{
		"type": se.Type,
		"package_version": se.PackageVersion,
		"index": se.Index,
		"payload": se.Payload,
		"blob": se.Blob,
	}
	if se.Step != nil {
		e.Put("step", se.Step.ToMap())
	}
	if se.Journey != nil {
		e.Put("journey", se.Journey.ToMap())
	}
	m := common.MapStr{"synthetics": e}
	if se.Error != nil {
		m["error"] = common.MapStr{
			"type": "synthetics",
			"message": se.Error,
		}
	}
	if se.URL != nil {
		u, e := url.Parse(*se.URL)
		if e != nil {
			m["url"] = wrappers.URLFields(u)
		} else {
			logp.Warn("Could not parse synthetics URL '%s'", se.URL)
		}
	}

	return m
}

type Step struct {
	Name string `json:"name"`
	Index int `json:"index"`
}

func (s *Step) ToMap() common.MapStr {
	return common.MapStr{
		"name": s.Name,
		"index": s.Index,
	}
}

type Journey struct {
	Name string `json:"name"`
	Id string `json:"id"`
}

func (j Journey) ToMap() common.MapStr {
	return common.MapStr{
		"name": j.Name,
		"id": j.Id,
	}
}

type RawResult struct {
	Journeys []map[string]interface{} `json:"journeys"`
}

type ExecMultiplexer struct {
	eventCounter *atomic.Int
	synthEvents  chan SynthEvent
	done         chan struct{}
}

func (e ExecMultiplexer) Close() {
	close(e.done)
	close(e.synthEvents)
}

func (e ExecMultiplexer) writeSynthEvent(se SynthEvent) {
	se.Index = e.eventCounter.Inc()
	e.synthEvents <- se
}

// SynthEvents returns a read only channel for synth events
func (e ExecMultiplexer) SynthEvents() <- chan SynthEvent {
	return e.synthEvents
}

// Done returns a channel that is closed when all output has been received
func (e ExecMultiplexer) Done() <- chan struct{} {
	return e.done
}

// Wait blocks until the multiplexer is done and has returned all data
func (e ExecMultiplexer) Wait() {
	<- e.done
}

func NewExecMultiplexer() *ExecMultiplexer {
	return &ExecMultiplexer{
		eventCounter: atomic.NewInt(-1), // Start from -1 so first call to Inc returns 0
		synthEvents: make(chan SynthEvent),
		done: make(chan struct{}),
	}
}
