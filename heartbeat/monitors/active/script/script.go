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

		var cmd *exec.Cmd
		if config.SuiteFile != "" {
			cmd = exec.Command(
				"node",
				config.SuiteFile,
				"-e", "production",
				"--json",
				"--headless",
			)
		} else {
			cmd = exec.Command(
				"npx",
				"elastic-synthetics",
				"--stdin",
				"--json",
				"--headless",
			)
		}



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

		stdoutLines, result := decodePipe(stdout)
		stderrLines, _ := decodePipe(stderr)

		processResult(event, result)

		eventext.MergeEventFields(event, common.MapStr{
			"script": common.MapStr{
				"stdout": stdoutLines,
				"stderr": stderrLines,
			},
		})

		if result != nil && len(result.Journeys) > 0 {
			eventext.MergeEventFields(event, common.MapStr{
				"script": common.MapStr{
					"journey": result.Journeys[0].Raw,
				},
			})
		}

		if err = cmd.Wait(); err != nil {
			return fmt.Errorf("error running cmd: %w", err)
		}

		return nil
	})

	return []jobs.Job{job}, 1, nil
}

func processResult(event *beat.Event, result *result) {
	if result == nil {
		logp.Warn("no result received!")
		return
	}
	if result.Journeys == nil || len(result.Journeys) == 0 {
		logp.Warn("result received with no journies: %#v", result.raw)
		return
	}

	journey := result.Journeys[0]
	status := "up"
	if journey.Error != nil {
		status = "down"
	}

	eventext.MergeEventFields(event, common.MapStr{
		"monitor": common.MapStr{
			"status": status,
			"duration.us": journey.Duration,
		},
	})

	u, err := url.Parse(journey.Url)
	if err != nil {
		logp.Warn("Could not parse journey URL %s", journey.Url)
	}

	eventext.MergeEventFields(event, common.MapStr{
		"url": wrappers.URLFields(u),
	})
}

func decodePipe(pipe io.ReadCloser) (lines []string, result *result) {
	pipeBio := bufio.NewReader(pipe)
	for {
		line, _, err := pipeBio.ReadLine()
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

func decodeResults(line []byte) (res *result, ok bool) {
	// We need to yield both a map[string]interface{} version of "Journeys" to pass through to ES
	// and a richer version that has accessible fields. Let's do both
	var rawRes = &rawResult{}
	res = &result{}

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

	res.raw = rawRes
	ok = true
	for idx, j := range res.Journeys {
		j.Raw = rawRes.Journeys[idx]
	}

	return
}

type result struct {
	formatVersion string `json:"format_version"`
	Journeys []*Journey `json:"journeys"`
	raw *rawResult
}

type rawResult struct {
	Journeys []map[string]interface{} `json:"journeys"`
}

type Journey struct {
	Url      string      `json:"url"`
	Steps    []Step      `json:"steps"`
	DataType string      `json:"__type__"`
	Error    interface{} `json:"error"`
	Duration interface{} `json:"elapsedMs"`
	Raw      map[string]interface{}
}

type Step struct {

}
