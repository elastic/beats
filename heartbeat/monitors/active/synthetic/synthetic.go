package synthetic

import (
	"bytes"
	"encoding/json"
	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"net/http"
)

func init() {
	monitors.RegisterActive("synthetic", create)
}

type synthproxyReq struct {
	Name string `json:"name"`
	Browser string `json:"browser"`
	Script string `json:"script"`
	ApiKey string `json:"api_key"`
}

func create(name string, cfg *common.Config) (js []jobs.Job, endpoints int, err error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, 0, err
	}

	for _, browser := range config.Browsers {
		sr := synthproxyReq{"TODO", browser, config.Script, config.ApiKey}
		srJson, err := json.Marshal(sr)
		if err != nil {
			return nil, 0, err
		}
		job := monitors.MakeSimpleCont(func(event *beat.Event) error {
			resp, err := http.Post(config.RunnerURL, "application/json", bytes.NewBuffer(srJson))
			if err != nil {
				return err
			}
			var decoded map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&decoded)
			if err != nil {
				return err
			}
			eventext.MergeEventFields(event, common.MapStr{
				"journey": decoded,
			})
			return nil
		})
		js = append(js, job)
	}

	return js, 1, nil
}
