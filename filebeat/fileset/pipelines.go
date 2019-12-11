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

package fileset

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/joeshaw/multierror"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// PipelineLoaderFactory builds and returns a PipelineLoader
type PipelineLoaderFactory func() (PipelineLoader, error)

// PipelineLoader is a subset of the Elasticsearch client API capable of loading
// the pipelines.
type PipelineLoader interface {
	LoadJSON(path string, json map[string]interface{}) ([]byte, error)
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
	GetVersion() common.Version
}

// MultiplePipelineUnsupportedError is an error returned when a fileset uses multiple pipelines but is
// running against a version of Elasticsearch that doesn't support this feature.
type MultiplePipelineUnsupportedError struct {
	module               string
	fileset              string
	esVersion            common.Version
	minESVersionRequired common.Version
}

func (m MultiplePipelineUnsupportedError) Error() string {
	return fmt.Sprintf(
		"the %s/%s fileset has multiple pipelines, which are only supported with Elasticsearch >= %s. Currently running with Elasticsearch version %s",
		m.module,
		m.fileset,
		m.minESVersionRequired.String(),
		m.esVersion.String(),
	)
}

// LoadPipelines loads the pipelines for each configured fileset.
func (reg *ModuleRegistry) LoadPipelines(esClient PipelineLoader, overwrite bool) error {
	for module, filesets := range reg.registry {
		for name, fileset := range filesets {
			// check that all the required Ingest Node plugins are available
			requiredProcessors := fileset.GetRequiredProcessors()
			logp.Debug("modules", "Required processors: %s", requiredProcessors)
			if len(requiredProcessors) > 0 {
				err := checkAvailableProcessors(esClient, requiredProcessors)
				if err != nil {
					return fmt.Errorf("Error loading pipeline for fileset %s/%s: %v", module, name, err)
				}
			}

			pipelines, err := fileset.GetPipelines(esClient.GetVersion())
			if err != nil {
				return fmt.Errorf("Error getting pipeline for fileset %s/%s: %v", module, name, err)
			}

			// Filesets with multiple pipelines can only be supported by Elasticsearch >= 6.5.0
			esVersion := esClient.GetVersion()
			minESVersionRequired := common.MustNewVersion("6.5.0")
			if len(pipelines) > 1 && esVersion.LessThan(minESVersionRequired) {
				return MultiplePipelineUnsupportedError{module, name, esVersion, *minESVersionRequired}
			}

			var pipelineIDsLoaded []string
			for _, pipeline := range pipelines {
				err = loadPipeline(esClient, pipeline.id, pipeline.contents, overwrite)
				if err != nil {
					err = fmt.Errorf("Error loading pipeline for fileset %s/%s: %v", module, name, err)
					break
				}
				pipelineIDsLoaded = append(pipelineIDsLoaded, pipeline.id)
			}

			if err != nil {
				// Rollback pipelines and return errors
				// TODO: Instead of attempting to load all pipelines and then rolling back loaded ones when there's an
				// error, validate all pipelines before loading any of them. This requires https://github.com/elastic/elasticsearch/issues/35495.
				errs := multierror.Errors{err}
				for _, pipelineID := range pipelineIDsLoaded {
					err = deletePipeline(esClient, pipelineID)
					if err != nil {
						errs = append(errs, err)
					}
				}
				return errs.Err()
			}
		}
	}
	return nil
}

func loadPipeline(esClient PipelineLoader, pipelineID string, content map[string]interface{}, overwrite bool) error {
	path := makeIngestPipelinePath(pipelineID)
	if !overwrite {
		status, _, _ := esClient.Request("GET", path, "", nil, nil)
		if status == 200 {
			logp.Debug("modules", "Pipeline %s already loaded", pipelineID)
			return nil
		}
	}

	err := setECSProcessors(esClient.GetVersion(), pipelineID, content)
	if err != nil {
		return fmt.Errorf("failed to adapt pipeline for ECS compatibility: %v", err)
	}

	body, err := esClient.LoadJSON(path, content)
	if err != nil {
		return interpretError(err, body)
	}
	logp.Info("Elasticsearch pipeline with ID '%s' loaded", pipelineID)
	return nil
}

// setECSProcessors sets required ECS options in processors when filebeat version is >= 7.0.0
// and ES is 6.7.X to ease migration to ECS.
func setECSProcessors(esVersion common.Version, pipelineID string, content map[string]interface{}) error {
	ecsVersion := common.MustNewVersion("7.0.0")
	if !esVersion.LessThan(ecsVersion) {
		return nil
	}

	p, ok := content["processors"]
	if !ok {
		return nil
	}
	processors, ok := p.([]interface{})
	if !ok {
		return fmt.Errorf("'processors' in pipeline '%s' expected to be a list, found %T", pipelineID, p)
	}

	minUserAgentVersion := common.MustNewVersion("6.7.0")
	for _, p := range processors {
		processor, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		if options, ok := processor["user_agent"].(map[string]interface{}); ok {
			if esVersion.LessThan(minUserAgentVersion) {
				return fmt.Errorf("user_agent processor requires option 'ecs: true', but Elasticsearch %v does not support this option (Elasticsearch %v or newer is required)", esVersion, minUserAgentVersion)
			}
			logp.Debug("modules", "Setting 'ecs: true' option in user_agent processor for field '%v' in pipeline '%s'", options["field"], pipelineID)
			options["ecs"] = true
		}
	}
	return nil
}

func deletePipeline(esClient PipelineLoader, pipelineID string) error {
	path := makeIngestPipelinePath(pipelineID)
	_, _, err := esClient.Request("DELETE", path, "", nil, nil)
	return err
}

func makeIngestPipelinePath(pipelineID string) string {
	return "/_ingest/pipeline/" + pipelineID
}

func interpretError(initialErr error, body []byte) error {
	var response struct {
		Error struct {
			RootCause []struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
				Header struct {
					ProcessorType string `json:"processor_type"`
				} `json:"header"`
				Index string `json:"index"`
			} `json:"root_cause"`
		} `json:"error"`
	}
	err := json.Unmarshal(body, &response)
	if err != nil {
		// this might be ES < 2.0. Do a best effort to check for ES 1.x
		var response1x struct {
			Error string `json:"error"`
		}
		err1x := json.Unmarshal(body, &response1x)
		if err1x == nil && response1x.Error != "" {
			return fmt.Errorf("The Filebeat modules require Elasticsearch >= 5.0. "+
				"This is the response I got from Elasticsearch: %s", body)
		}

		return fmt.Errorf("couldn't load pipeline: %v. Additionally, error decoding response body: %s",
			initialErr, body)
	}

	// missing plugins?
	if len(response.Error.RootCause) > 0 &&
		response.Error.RootCause[0].Type == "parse_exception" &&
		strings.HasPrefix(response.Error.RootCause[0].Reason, "No processor type exists with name") &&
		response.Error.RootCause[0].Header.ProcessorType != "" {

		return fmt.Errorf("This module requires an Elasticsearch plugin that provides the %s processor. "+
			"Please visit the Elasticsearch documentation for instructions on how to install this plugin. "+
			"Response body: %s", response.Error.RootCause[0].Header.ProcessorType, body)

	}

	// older ES version?
	if len(response.Error.RootCause) > 0 &&
		response.Error.RootCause[0].Type == "invalid_index_name_exception" &&
		response.Error.RootCause[0].Index == "_ingest" {

		return fmt.Errorf("The Ingest Node functionality seems to be missing from Elasticsearch. "+
			"The Filebeat modules require Elasticsearch >= 5.0. "+
			"This is the response I got from Elasticsearch: %s", body)
	}

	return fmt.Errorf("couldn't load pipeline: %v. Response body: %s", initialErr, body)
}
