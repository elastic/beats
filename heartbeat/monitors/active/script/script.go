package script

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"io"
	"net/url"
	"os/exec"
	"os/user"
)

func init() {
	monitors.RegisterActive("script", create)
	monitors.RegisterActive("synthetic/script", create)
}

func create(name string, cfg *common.Config) (js []jobs.Job, endpoints int, err error) {
	curUser, err := user.Current()
	if err != nil {
		return nil, 0, fmt.Errorf("could not determine current user for script monitor %w: ", err)
	}
	if curUser.Uid == "0" {
		return nil, 0, fmt.Errorf("script monitors cannot be run as root! Current UID is %s", curUser.Uid)
	}

	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, 0, err
	}

	job := monitors.MakeSimpleCont(func(event *beat.Event) error {
		logp.Info("Start script job")
		cmd := exec.Command(
			"npx",
			"elastic-synthetics",
			"run",
			"--stdin",
			"--json",
			"--headless",
		)

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

		stderrResults := decodePipe(stderr)

		var stdoutResults []string
		var rawResults map[string]interface{}
		for _, line := range decodePipe(stdout) {
			if s, ok := line.(string); ok {
				logp.Warn("STRING HERE")
				stdoutResults = append(stdoutResults, s)
			} else if r, ok := line.(Results); ok {
				status := "down"
				rawResults = r.Raw

				if len(r.Journeys) > 0 {
					journey := r.Journeys[0]
					if journey.error != nil {
						status = "down"
					}
					logp.Warn("RESULTS HERE %#v",  r)

					eventext.MergeEventFields(event, common.MapStr{
						"monitor": common.MapStr{
							"status": status,
							"duration.us": journey.duration,
						},
					})

					u, err := url.Parse(journey.Url)
					if err != nil {
						eventext.MergeEventFields(event, common.MapStr{
							"url": wrappers.URLFields(u),
						})
					}
				}
			}
		}


		eventext.MergeEventFields(event, common.MapStr{
			"script": common.MapStr{
				"stdout": stdoutResults,
				"stderr": stderrResults,
				"journey": rawResults,
			},
		})

		if err = cmd.Wait(); err != nil {
			return fmt.Errorf("error running cmd: %w", err)
		}

		return nil
	})

	return []jobs.Job{job}, 1, nil
}

func decodePipe(pipe io.ReadCloser) []interface{} {
	pipeBio := bufio.NewReader(pipe)
	var results []interface{}
	for {
		line, _, err := pipeBio.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			logp.Warn("error reading line: %w", err)
		}

		var decoded Results
		err = json.Unmarshal(line, &decoded)
		var raw map[string]interface{}
		json.Unmarshal(line, &raw)
		decoded.Raw = raw
		logp.Warn("> %s", string(line))
		if err != nil {
			results = append(results, line)
		}

		results = append(results, decoded)
	}

	return results
}

type Results struct {
	formatVersion string `json:"format_version"`
	Journeys []Journey `json:"journeys"`
	Raw common.MapStr
}

type Journey struct {
	Url   string `json:"url"`
	Steps []Step `json:"steps"`
	dataType string `json:"__type__"`
	error interface{} `json:"error"`
	duration interface{} `json:"elapsedMs"`
}

type Step struct {

}
