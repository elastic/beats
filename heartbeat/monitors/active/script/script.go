package script

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"io"
	"os/exec"
)

func init() {
	monitors.RegisterActive("script", create)
	monitors.RegisterActive("synthetic/script", create)
}

type synthproxyReq struct {
	ScriptParams map[string]interface{} `json:"script_params"`
	Script string `json:"script"`
}

func create(name string, cfg *common.Config) (js []jobs.Job, endpoints int, err error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, 0, err
	}

	job := monitors.MakeSimpleCont(func(event *beat.Event) error {
		logp.Info("Start script job")
		cmd := exec.Command(
			"node",
			"/home/andrewvc/projects/synthetic-monitoring/elastic-synthetics/dist",
			"run",
			"--stdin",
			"--json",
		)

		logp.Warn("EXEC %s",cmd.String())

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("could not attach stdout: %w", err)
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("could not attach stderr: %w", err)
		}

		stdin, err := cmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("could not attach stdin: %w", err)
		}

		if err = cmd.Start(); err != nil {
			return fmt.Errorf("Could not start cmd: %w", err)
		}

		_, err = stdin.Write([]byte(config.Script))
		if err != nil {
			return fmt.Errorf("could not write to script stdin: %w", err)
		}
		stdin.Close()

		stdoutResults := decodePipe(stdout)
		stderrResults := decodePipe(stderr)

		eventext.MergeEventFields(event, map[string]interface{}{
			"script.stdout": stdoutResults,
			"script.stderr": stderrResults,
		})

		if err = cmd.Wait(); err != nil {
			return fmt.Errorf("error running cmd: %w", err)
		}

		return nil
	})

	return []jobs.Job{job}, 1, nil
}


func decodePipe(pipe io.ReadCloser) []map[string]interface{} {
	pipeBio := bufio.NewReader(pipe)
	var results []map[string]interface{}
	for {
		line, _, err := pipeBio.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			logp.Warn("error reading line: %w", err)
		}

		var decoded map[string]interface{}
		err = json.Unmarshal(line, &decoded)
		if err != nil {
			decoded = map[string]interface{}{"message": string(line)}
		}

		results = append(results, decoded)
	}

	return results
}
