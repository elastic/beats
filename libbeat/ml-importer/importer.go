// Package mlimporter contains code for loading Elastic X-Pack Machine Learning job configurations.
package mlimporter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/pkg/errors"
)

// MLConfig contains the required configuration for loading one job and the associated
// datafeed.
type MLConfig struct {
	ID           string `config:"id"`
	JobPath      string `config:"job"`
	DatafeedPath string `config:"datafeed"`
}

// MLLoader is a subset of the Elasticsearch client API capable of
// loading the ML configs.
type MLLoader interface {
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
	LoadJSON(path string, json map[string]interface{}) ([]byte, error)
}

func readJSONFile(path string) (common.MapStr, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var result common.MapStr
	err = json.Unmarshal(file, &result)
	return result, err
}

// ImportMachineLearningJob uploads the job and datafeed configuration to ES/xpack.
func ImportMachineLearningJob(esClient MLLoader, cfg *MLConfig) error {
	jobURL := fmt.Sprintf("/_xpack/ml/anomaly_detectors/%s", cfg.ID)
	datafeedURL := fmt.Sprintf("/_xpack/ml/datafeeds/datafeed-%s", cfg.ID)

	// We always overwrite ML job configs, so delete them before loading
	status, response, err := esClient.Request("GET", jobURL, "", nil, nil)
	if status == 200 {
		logp.Debug("machine-learning", "Job %s already exists", cfg.ID)
		return nil
	}
	if status != 404 && err != nil {
		return errors.Errorf("Error checking that job exists: %v. Response %s", err, response)
	}

	job, err := readJSONFile(cfg.JobPath)
	if err != nil {
		return errors.Errorf("Error reading job file %s: %v", cfg.JobPath, err)
	}

	body, err := esClient.LoadJSON(jobURL, job)
	if err != nil {
		return errors.Wrapf(err, "load job under %s. Response body: %s", jobURL, body)
	}

	datafeed, err := readJSONFile(cfg.DatafeedPath)
	if err != nil {
		return errors.Errorf("Error reading datafeed path %s: %v", cfg.DatafeedPath, err)
	}
	// set the job ID
	datafeed.Put("job_id", cfg.ID)

	body, err = esClient.LoadJSON(datafeedURL, datafeed)
	if err != nil {
		return errors.Wrapf(err, "load datafeed under %s. Response body: %s", datafeedURL, body)
	}

	return nil
}

// HaveXpackML checks whether X-pack is installed and has Machine Learning enabled.
func HaveXpackML(esClient MLLoader) (bool, error) {

	status, response, err := esClient.Request("GET", "/_xpack", "", nil, nil)
	if status == 404 || status == 400 {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	type xpackResponse struct {
		Features struct {
			ML struct {
				Available bool `json:"available"`
				Enabled   bool `json:"enabled"`
			} `json:"ml"`
		} `json:"features"`
	}
	var xpack xpackResponse
	err = json.Unmarshal(response, &xpack)
	if err != nil {
		return false, errors.Wrap(err, "unmarshal")
	}
	return xpack.Features.ML.Available && xpack.Features.ML.Enabled, nil
}
