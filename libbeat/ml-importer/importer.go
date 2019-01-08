// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Package mlimporter contains code for loading Elastic X-Pack Machine Learning job configurations.
package mlimporter

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

var (
	esDataFeedURL        = "/_xpack/ml/datafeeds/datafeed-%s"
	esJobURL             = "/_xpack/ml/anomaly_detectors/%s"
	kibanaGetModuleURL   = "/api/ml/modules/get_module/%s"
	kibanaRecognizeURL   = "/api/ml/modules/recognize/%s"
	kibanaSetupModuleURL = "/api/ml/modules/setup/%s"
)

// MLConfig contains the required configuration for loading one job and the associated
// datafeed.
type MLConfig struct {
	ID           string `config:"id"`
	JobPath      string `config:"job"`
	DatafeedPath string `config:"datafeed"`
	MinVersion   string `config:"min_version"`
}

// MLLoader is a subset of the Elasticsearch client API capable of
// loading the ML configs.
type MLLoader interface {
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
	LoadJSON(path string, json map[string]interface{}) ([]byte, error)
	GetVersion() common.Version
}

// MLSetupper is a subset of the Kibana client API capable of setting up ML objects.
type MLSetupper interface {
	Request(method, path string, params url.Values, headers http.Header, body io.Reader) (int, []byte, error)
	GetVersion() common.Version
}

// MLResponse stores the relevant parts of the response from Kibana to check for errors.
type MLResponse struct {
	Datafeeds []struct {
		ID      string
		Success bool
		Error   struct {
			Msg string
		}
	}
	Jobs []struct {
		ID      string
		Success bool
		Error   struct {
			Msg string
		}
	}
	Kibana struct {
		Dashboard []struct {
			Success bool
			ID      string
			Exists  bool
			Error   struct {
				Message string
			}
		}
		Search []struct {
			Success bool
			ID      string
			Exists  bool
			Error   struct {
				Message string
			}
		}
		Visualization []struct {
			Success bool
			ID      string
			Exists  bool
			Error   struct {
				Message string
			}
		}
	}
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
	jobURL := fmt.Sprintf(esJobURL, cfg.ID)
	datafeedURL := fmt.Sprintf(esDataFeedURL, cfg.ID)

	if len(cfg.MinVersion) > 0 {
		esVersion := esClient.GetVersion()
		if !esVersion.IsValid() {
			return errors.New("Invalid Elasticsearch version")
		}

		minVersion, err := common.NewVersion(cfg.MinVersion)
		if err != nil {
			return errors.Errorf("Error parsing min_version: %s: %v", minVersion, err)
		}

		if esVersion.LessThan(minVersion) {
			logp.Debug("machine-learning", "Skipping job %s, because ES version (%s) is smaller than min version (%s)",
				cfg.ID, esVersion.String(), minVersion)
			return nil
		}
	}

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

// SetupModule creates ML jobs, data feeds and dashboards for modules.
func SetupModule(kibanaClient MLSetupper, module, prefix string) error {
	setupURL := fmt.Sprintf(kibanaSetupModuleURL, module)
	prefixPayload := fmt.Sprintf("{\"prefix\": \"%s\"}", prefix)
	status, response, err := kibanaClient.Request("POST", setupURL, nil, nil, strings.NewReader(prefixPayload))
	if status != 200 {
		return errors.Errorf("cannot set up ML with prefix: %s", prefix)
	}
	if err != nil {
		return err
	}

	return checkResponse(response)
}

func checkResponse(r []byte) error {
	var errs multierror.Errors

	var resp MLResponse
	err := json.Unmarshal(r, &resp)
	if err != nil {
		return err
	}

	for _, feed := range resp.Datafeeds {
		if !feed.Success {
			if strings.HasPrefix(feed.Error.Msg, "[status_exception] A datafeed") || strings.HasPrefix(feed.Error.Msg, "[resource_already_exists_exception]") {
				logp.Debug("machine-learning", "Datafeed already exists: %s, error: %s", feed.ID, feed.Error.Msg)
				continue
			}
			errs = append(errs, errors.Errorf(feed.Error.Msg))
		}
	}
	for _, job := range resp.Jobs {
		if strings.HasPrefix(job.Error.Msg, "[resource_already_exists_exception]") {
			logp.Debug("machine-learning", "Job already exists: %s, error: %s", job.ID, job.Error.Msg)
			continue
		}
		if !job.Success {
			errs = append(errs, errors.Errorf(job.Error.Msg))
		}
	}
	for _, dashboard := range resp.Kibana.Dashboard {
		if !dashboard.Success {
			if dashboard.Exists || strings.Contains(dashboard.Error.Message, "version conflict, document already exists") {
				logp.Debug("machine-learning", "Dashboard already exists: %s, error: %s", dashboard.ID, dashboard.Error.Message)
			} else {
				errs = append(errs, errors.Errorf("error while setting up dashboard: %s", dashboard.ID))
			}
		}
	}
	for _, search := range resp.Kibana.Search {
		if !search.Success {
			if search.Exists || strings.Contains(search.Error.Message, "version conflict, document already exists") {
				logp.Debug("machine-learning", "Search already exists: %s", search.ID)
			} else {
				errs = append(errs, errors.Errorf("error while setting up search: %s, error: %s", search.ID, search.Error.Message))
			}
		}
	}
	for _, visualization := range resp.Kibana.Visualization {
		if !visualization.Success {
			if visualization.Exists || strings.Contains(visualization.Error.Message, "version conflict, document already exists") {
				logp.Debug("machine-learning", "Visualization already exists: %s", visualization.ID)
			} else {
				errs = append(errs, errors.Errorf("error while setting up visualization: %s, error: %s", visualization.ID, visualization.Error.Message))
			}
		}
	}

	return errs.Err()
}
