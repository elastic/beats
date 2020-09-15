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
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"
)

func ListJourneys(ctx context.Context, suiteFile string, params common.MapStr) (journeyNames []string, err error) {
	cmd := exec.Command(
		"node",
		suiteFile,
		"-e", "production",
		"--json",
		"--headless",
		"--screenshots",
		"--dry-run",
	)

	mpx, err := runCmd(ctx, cmd, nil, params)
	for {
		select {
		case se := <- mpx.SynthEvents():
			if se.Type == "journey/start" {
				journeyNames = append(journeyNames, se.Journey.Name)
			}
		case <- mpx.Done():
			break
		}
	}

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
			event.Timestamp = se.Timestamp
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
	defer resultsWriter.Close()
	defer resultsReader.Close()

	// Common args
	cmd.Args = append(cmd.Args,
		"--outfd", fmt.Sprintf("%d", resultsWriter.Fd()),
		"--json",
		"--headless",
	)
	if len(params) > 0 {
		paramsBytes, _ := json.Marshal(params)
		cmd.Args = append(cmd.Args, "--suite-params", string(paramsBytes))
	}

	logp.Info("Running command: %s", cmd.String())

	if stdinStr	!= nil {
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
		sendResults(resultsReader, mpx.writeSynthEvent)
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
		logp.Warn("error executing command '%s': %w", cmd.String(), err)
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
			logp.Warn("Error encountered scanning console line: %s. Line was %s", scanner.Err(), scanner.Text())
		}
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

func sendResults(rdr io.Reader, cb func(SynthEvent)) error {
	scanner := bufio.NewScanner(rdr)
	buf := make([]byte, 1024*1024*2) // 2MiB initial buffer
	scanner.Buffer(buf, 1024*1024*200) // Max 200MiB Buffer
	for scanner.Scan() {
		if scanner.Err() != nil {
			return scanner.Err()
		}

		var res = SynthEvent{}
		err := json.Unmarshal(scanner.Bytes(), &res)
		if err != nil {
			return err
		}

		if cb != nil {
			cb(res)
		}
	}
	return nil
}

type SynthEvent struct {
	Type           string                 `json:"type"`
	PackageVersion string                 `json:"package_version"`
	Index          int                    `json:"index""`
	Journey        SynthJourney           `json:"journey"`
	Timestamp      time.Time              `json:"@timestamp"`
	Payload        map[string]interface{} `json:"payload"`
}

func (se SynthEvent) ToMap() common.MapStr {
	// We don't add @timestamp to the map string since that's specially handled in beat.Event
	return common.MapStr{
		"type": se.Type,
		"package_version": se.PackageVersion,
		"index": se.Index,
		"journey": se.Journey.ToMap(),
		"payload": se.Payload,
	}
}

type SynthJourney struct {
	Name string `json:"name"`
	Id string `json:"id"`
}

func (sj SynthJourney) ToMap() common.MapStr {
	return common.MapStr{
		"name": sj.Name,
		"id": sj.Id,
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
