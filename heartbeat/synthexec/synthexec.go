// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package synthexec

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"
)

func ListJourneys(ctx context.Context, suiteFile string) (journeyNames []string, err error) {
	cmd := exec.Command(
		"node",
		suiteFile,
		"-e", "production",
		"--json",
		"--headless",
		"--screenshots",
		"--dry-run",
	)

	handler := NewExecHandler()
	handler.OnResult = func(result Result) {
		if result.Type == "journey/start" {
			journeyNames = append(journeyNames, result.Journey.Name)
		}
	}

	err = runCmd(ctx, cmd, nil, handler)
	if err != nil {
		return nil, err
	}

	select {
		case <- handler.Done:
		case <- ctx.Done():
	}

	return journeyNames, nil
}

func RunSuite(ctx context.Context, suiteFile string, journeyName string, handler *ExecHandler) error {
	logp.Warn("RUNNING JOURNEY %s", journeyName)
	cmd := exec.Command(
		"node",
		suiteFile,
		"-e", "production",
		"--json",
		"--headless",
		"--screenshots",
		"--journey-name", journeyName,
	)

	return runCmd(ctx, cmd, nil, handler)
}

func RunScript(ctx context.Context, script string, handler *ExecHandler) (err error) {
	cmd := exec.Command(
		"npx",
		"@elastic/synthetics",
		"--stdin",
		"--json",
		"--headless",
		"--screenshots",
	)

	return runCmd(ctx, cmd, &script, handler)
}

func runCmd(
	ctx context.Context,
	cmd *exec.Cmd,
	stdinStr *string,
	handler *ExecHandler,
	) error {
	if handler.OnDone != nil {
		defer handler.OnDone()
	}
	if handler.Done != nil {
		defer func() {handler.Done <- struct {}{}}()
	}
	// Setup a pipe for structured output
	resultsReader, resultsWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	defer resultsWriter.Close()
	defer resultsReader.Close()

	cmd.Args = append(cmd.Args, "--outfd", fmt.Sprintf("%d", resultsWriter.Fd()))

	logp.Info("Running command: %s", cmd.String())

	if stdinStr	!= nil {
		cmd.Stdin = strings.NewReader(*stdinStr)
	}

	// Handle the console output
	lineCounter := atomic.MakeInt(0)
	// Send stdout into the output
	stdoutPipe, err := cmd.StdoutPipe()
	go sendConsoleLines(stdoutPipe, "stdout", lineCounter, handler.OnConsole)
	// Send stderr into the output
	stderrPipe, err := cmd.StderrPipe()
	go sendConsoleLines(stderrPipe, "stderr", lineCounter,  handler.OnConsole)

	// Send the test results into the output
	go sendResults(resultsReader, handler.OnResult)

	err = cmd.Start()
	if err != nil {
		return err
	}

	// Kill the process if the context ends
	go func() {
		select {
		case <- ctx.Done():
			cmd.Process.Kill()
		}
	}()

	cmd.Wait()

	return nil
}

func sendConsoleLines(rdr io.Reader, typ string, lineCounter atomic.Int, cb func(line ConsoleLine)) {
	scanner := bufio.NewScanner(rdr)
	buf := make([]byte, 1024*1024*2) // 2MiB initial buffer
	scanner.Buffer(buf, 1024*1024*200) // Max 200MiB Buffer
	for scanner.Scan() {
		if scanner.Err() != nil {
			logp.Warn("Error encountered scanning console line: %s. Line was %s", scanner.Err(), scanner.Text())
		}
		if cb != nil {
			cb(ConsoleLine{
				Type:    typ,
				Message: scanner.Text(),
				Number:  lineCounter.Inc(),
			})
		}
	}
}

func sendResults(rdr io.Reader, cb func(Result)) error {
	scanner := bufio.NewScanner(rdr)
	buf := make([]byte, 1024*1024*2) // 2MiB initial buffer
	scanner.Buffer(buf, 1024*1024*200) // Max 200MiB Buffer
	for scanner.Scan() {
		if scanner.Err() != nil {
			return scanner.Err()
		}

		var res = Result{}
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

type ConsoleLine struct {
	Message string `json:"message"`
	Number int `json:"index"`
	Type string `json:"type"`
}

type Result struct {
	Type string `json:"type"`
	PackageVersion string `json:"package_version"`
	Journey ResultJourney `json:"journey"`
	Timestamp time.Time `json:"@timestamp"`
	Payload map[string]interface{} `json:"payload"`
}

type ResultJourney struct {
	Name string `json:"name"`
	Id string `json:"id"`
}

type RawResult struct {
	Journeys []map[string]interface{} `json:"journeys"`
}

type ExecHandler struct {
	OnConsole func(line ConsoleLine)
	OnResult func(result Result)
	OnDone func()
	Done chan struct{}
}

func NewExecHandler() *ExecHandler {
	return &ExecHandler{
		Done: make(chan struct{}),
	}
}
