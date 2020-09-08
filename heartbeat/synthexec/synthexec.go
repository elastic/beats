package synthexec

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/elastic/beats/v7/libbeat/logp"
	"io"
	"os/exec"
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

func runCmd(cmd *exec.Cmd, stdinStr *string) (out *CmdOut, err error) {
	logp.Info("Running command: %s", cmd.String())

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("could not attach stdout: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("could not attach stderr: %w", err)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("could not attach stdin: %w", err)
	}

	if err = cmd.Start(); err != nil {
		return nil, fmt.Errorf("Could not start cmd: %w", err)
	}

	if stdinStr	!= nil {
		_, err = stdin.Write([]byte(*stdinStr))
	}
	if err != nil {
		return nil, fmt.Errorf("could not write to script stdin: %w", err)
	}
	stdin.Close()

	stdoutLines, result := decodePipe(stdout)
	stderrLines, _ := decodePipe(stderr)

	if err = cmd.Wait(); err != nil {
		for _, line := range stdoutLines {
			logp.Warn("stdout: %s", line)
		}
		for _, line := range stderrLines {
			logp.Warn("stderr: %s", line)
		}
		return nil, fmt.Errorf("error running cmd: %w", err)
	}

	return &CmdOut{
		Result: result,
		Stdout: stdoutLines,
		Stderr: stderrLines,
	}, nil
}

func decodePipe(pipe io.ReadCloser) (lines []string, result *Result) {
	pipeBio := bufio.NewReader(pipe)
	for {
		line, err := pipeBio.ReadBytes([]byte("\n")[0])

		if err == io.EOF {
			break
		} else if err != nil {
			logp.Warn("error reading line: %w", err)
		}

		res, ok := decodeResults(line)
		if ok { // append the rich results if that's what this line is
			result = res
		} else { // otherwise just append
			lines = append(lines, string(line))
		}
	}

	return
}

func decodeResults(line []byte) (res *Result, ok bool) {
	// We need to yield both a map[string]interface{} version of "Journeys" to pass through to ES
	// and a richer version that has accessible fields. Let's do both
	var rawRes = &RawResult{}
	res = &Result{}

	err := json.Unmarshal(line, rawRes)
	// This must just be a plain line
	if err != nil {
		return
	}

	err = json.Unmarshal(line, res)
	if err != nil {
		logp.Warn("Raw result decoded successfully, but richer one did not: %s", line)
		return
	}

	res.Raw = rawRes
	ok = true
	for idx, j := range res.Journeys {
		j.Raw = rawRes.Journeys[idx]
	}

	return
}

type CmdOut struct {
	Result *Result
	Stdout []string
	Stderr []string
}

type Result struct {
	formatVersion string `json:"format_version"`
	Journeys []*Journey `json:"journeys"`
	Raw *RawResult
}

type RawResult struct {
	Journeys []map[string]interface{} `json:"journeys"`
}

type Journey struct {
	Name string `json:"name"`
	Url      string      `json:"url"`
	Steps    []Step      `json:"steps"`
	DataType string      `json:"__type__"`
	Error    interface{} `json:"error"`
	Duration interface{} `json:"elapsedMs"`
	Raw      map[string]interface{}
	Status string `json:"status"`
}

type Step struct {
	Screenshot string `json:"screenshot"`
}
