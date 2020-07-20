package synthetic

import (
	"bytes"
	"encoding/json"
	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"io/ioutil"
	"net/http"
)

func init() {
	monitors.RegisterActive("synthetic", create)
}

type synthproxyReq struct {
	Name string `json:"name"`
	Browser string `json:"browser"`
	ScriptParams map[string]interface{} `json:"script_params"`
	Script string `json:"script"`
	ApiKey string `json:"api_key"`
}

func create(name string, cfg *common.Config) (js []jobs.Job, endpoints int, err error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, 0, err
	}

	for _, browser := range config.Browsers {
		sr := synthproxyReq{
			Name: "TODO",
			Browser: browser,
			Script: config.Script,
			ScriptParams: config.ScriptParams,
			ApiKey: config.ApiKey,
		}
		srJson, err := json.Marshal(sr)
		if err != nil {
			return nil, 0, err
		}
		job := monitors.MakeSimpleCont(func(event *beat.Event) error {
			logp.Info("START JOB")
			resp, err := http.Post(config.RunnerURL, "application/json", bytes.NewBuffer(srJson))
			if err != nil {
				return err
			}

			bodyStr, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				logp.Warn("read err", err)
			}

			var decoded map[string]interface{}
			err = json.Unmarshal(bodyStr, &decoded)
			if err != nil {
				logp.Warn("DECODE ERR %v. Body is '%v'", err)
				return err
			}
			eventext.MergeEventFields(event, common.MapStr{
				"journey": decoded,
			})

			if v, _ := decoded["error"]; v != nil {
				logp.Warn("Error is: %v", v)
				return errors.New(v)
			}

			logp.Warn("COMPLETE")

			return nil
		})
		js = append(js, job)
	}

	return js, 1, nil
}
