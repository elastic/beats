// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package synthexec

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/elastic/beats/v7/libbeat/logp"
)

func ListSuite(suiteFile string) (out *CmdOut, err error) {
	cmd := exec.Command(
		"node",
		suiteFile,
		"-e", "production",
		"--json",
		"--headless",
		"--screenshots",
		"--dry-run",
	)

	return runCmd(cmd, nil)
}

func RunSuite(suiteFile string, journeyName string) (out *CmdOut, err error) {
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

	return runCmd(cmd, nil)
}

func RunScript(script string) (out *CmdOut, err error) {
	cmd := exec.Command(
		"npx",
		"elastic-synthetics",
		"--stdin",
		"--json",
		"--headless",
		"--screenshots",
	)

	return runCmd(cmd, &script)
}

func runCmd(cmd *exec.Cmd, stdinStr *string) (*CmdOut, error) {
	logp.Info("Running command: %s", cmd.String())

	if stdinStr	!= nil {
		cmd.Stdin = strings.NewReader(*stdinStr)
	}

	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	out := &CmdOut{}
	scanner := bufio.NewScanner(bytes.NewReader(outBytes))
	buf := make([]byte, 1024*1024*2) // 2MiB initial buffer
	scanner.Buffer(buf, 1024*1024*200) // Max 200MiB Buffer
	for scanner.Scan() {
		if scanner.Err() != nil {
			logp.Warn("GOT SCAN ERR %w", scanner.Err())
			return nil, scanner.Err()
		}

		var result *Result
		if out.Result == nil {
			result = decodeResult(scanner.Bytes())
			out.Result = result
		}
		if result != nil {
			logp.Warn("GOT RESULT %s", result)
		}
		if result == nil {
			logp.Warn("GOT LINE '%s'", scanner.Text())
			out.Lines = append(out.Lines, scanner.Text())
		}
	}

	return out, err
}

// decodeResult attempts to decode the given line to our result type, returns nil if invalid.
func decodeResult(line []byte) (*Result) {
	// We need to yield both a map[string]interface{} version of "Journeys" to pass through to ES
	// and a richer version that has accessible fields. Let's do both
	var rawRes = &RawResult{}
	res := &Result{}

	err := json.Unmarshal(line, rawRes)
	// This must just be a plain line
	if err != nil {
		return nil
	}

	err = json.Unmarshal(line, res)
	if err != nil {
		logp.Warn("Raw result decoded successfully, but richer one did not: %s", line)
		return nil
	}

	res.Raw = rawRes
	for idx, j := range res.Journeys {
		j.Raw = rawRes.Journeys[idx]
	}

	return res
}

type CmdOut struct {
	Result *Result
	Lines []string
}

type Result struct {
	formatVersion string     `json:"format_version"`
	Journeys      []*Journey `json:"journeys"`
	Raw           *RawResult
}

type RawResult struct {
	Journeys []map[string]interface{} `json:"journeys"`
}

type Journey struct {
	Name     string      `json:"name"`
	Url      string      `json:"url"`
	Steps    []Step      `json:"steps"`
	DataType string      `json:"__type__"`
	Error    interface{} `json:"error"`
	Duration interface{} `json:"elapsedMs"`
	Raw      map[string]interface{}
	Status   string `json:"status"`
}

type Step struct {
	Screenshot string `json:"screenshot"`
}
