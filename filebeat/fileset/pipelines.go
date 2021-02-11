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
	"errors"
	"fmt"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/joeshaw/multierror"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
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

// processorCompatibility defines a single processors minimum version requirements.
type processorCompatibility struct {
	minVersion           *common.Version
	name                 string
	makeConfigCompatible func(log *logp.Logger, processor map[string]interface{}) error
}

var processorCompatibilityChecks = []processorCompatibility{
	{
		name:                 "uri_parts",
		minVersion:           common.MustNewVersion("7.12.0"),
		makeConfigCompatible: nil,
	},
	{
		name:                 "set",
		minVersion:           common.MustNewVersion("7.9.0"),
		makeConfigCompatible: modifySetProcessor,
	},
	{
		name:                 "append",
		minVersion:           common.MustNewVersion("7.10.0"),
		makeConfigCompatible: modifyAppendProcessor,
	},
	{
		name:                 "user_agent",
		minVersion:           common.MustNewVersion("6.7.0"),
		makeConfigCompatible: setECSProcessors,
	},
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
	spew.Dump(content)
	err := setProcessors(esClient.GetVersion(), pipelineID, content)
	if err != nil {
		return fmt.Errorf("Failed to adapt pipeline with backwards compatibility changes: %w", err)
	}

	body, err := esClient.LoadJSON(path, content)
	if err != nil {
		return interpretError(err, body)
	}
	logp.Info("Elasticsearch pipeline with ID '%s' loaded", pipelineID)
	return nil
}

// setProcessors iterates over all configured processors and performs the
// function related to it. If no function is set, it will delete the processor if
// the version of ES is under the required version number.
func setProcessors(esVersion common.Version, pipelineID string, content map[string]interface{}) error {
	log := logp.NewLogger("fileset").With("pipeline", pipelineID)
	p, ok := content["processors"]
	if !ok {
		return nil
	}

	processors, ok := p.([]interface{})
	if !ok {
		return fmt.Errorf("'processors' in pipeline '%s' expected to be a list, found %T", pipelineID, p)
	}

	// A list of all processor names and versions to be checked.

	var newProcessors []interface{}
	var appendProcessor bool
	for i, p := range processors {
		appendProcessor = true
		processor, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		for _, proc := range processorCompatibilityChecks {
			_, found := processor[proc.name]
			if !found {
				continue
			}

			if options, ok := processor[proc.name].(map[string]interface{}); ok {
				if !esVersion.LessThan(proc.minVersion) {
					if proc.name == "user_agent" {
						logp.Debug("modules", "Setting 'ecs: true' option in user_agent processor for field '%v' in pipeline '%s'", options["field"], pipelineID)
						options["ecs"] = true
					}
					continue
				}

				if proc.makeConfigCompatible != nil {
					if err := proc.makeConfigCompatible(log.With("processor_type", proc.name, "processor_index", i), processor); err != nil {
						return err
					}
				} else {
					appendProcessor = false
				}
			}
		}
		if appendProcessor {
			newProcessors = append(newProcessors, processors[i])
		}

	}
	content["processors"] = newProcessors
	return nil
}

// setECSProcessors sets required ECS options in processors when filebeat version is >= 7.0.0
// and ES is 6.7.X to ease migration to ECS.
func setECSProcessors(log *logp.Logger, processor map[string]interface{}) error {
	return errors.New("user_agent processor requires option 'ecs: true', Elasticsearch 6.7 or newer required")
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

// modifySetProcessor replaces ignore_empty_value option with an if statement
// so ES less than 7.9 will still work
func modifySetProcessor(log *logp.Logger, processor map[string]interface{}) error {
	options, ok := processor["set"].(map[string]interface{})

	if !ok {
		return nil
	}
	_, ok = options["ignore_empty_value"].(bool)
	if !ok {
		// don't have ignore_empty_value nothing to do
		return nil
	}

	log.Debug("Removing unsupported 'ignore_empty_value' in set processor")
	delete(options, "ignore_empty_value")

	_, ok = options["if"].(string)
	if ok {
		// assume if check is sufficient
		return nil
	}
	val, ok := options["value"].(string)
	if !ok {
		return nil
	}

	newIf := strings.TrimLeft(val, "{ ")
	newIf = strings.TrimRight(newIf, "} ")
	newIf = strings.ReplaceAll(newIf, ".", "?.")
	newIf = "ctx?." + newIf + " != null"

	log.Debug("adding if %s to replace 'ignore_empty_value' in set processor", newIf)
	options["if"] = newIf

	return nil
}

// modifyAppendProcessor replaces allow_duplicates option with an if statement
// so ES less than 7.10 will still work
func modifyAppendProcessor(log *logp.Logger, processor map[string]interface{}) error {
	options, ok := processor["append"].(map[string]interface{})
	if !ok {
		return nil
	}
	allow, ok := options["allow_duplicates"].(bool)

	if !ok {
		// don't have allow_duplicates, nothing to do
		return nil
	}

	log.Debug("removing unsupported 'allow_duplicates' in append processor")
	delete(options, "allow_duplicates")
	if allow {
		// it was set to true, nothing else to do after removing the option
		return nil
	}

	currIf, _ := options["if"].(string)
	if strings.Contains(strings.ToLower(currIf), "contains") {
		// if it has a contains statement, we assume it is checking for duplicates already
		return nil
	}
	field, ok := options["field"].(string)
	if !ok {
		return nil
	}
	val, ok := options["value"].(string)
	if !ok {
		return nil
	}

	field = strings.ReplaceAll(field, ".", "?.")

	val = strings.TrimLeft(val, "{ ")
	val = strings.TrimRight(val, "} ")
	val = strings.ReplaceAll(val, ".", "?.")

	if currIf == "" {
		// if there is not a previous if we add a value sanity check
		currIf = fmt.Sprintf("ctx?.%s != null", val)
	}

	newIf := fmt.Sprintf("%s && ((ctx?.%s instanceof List && !ctx?.%s.contains(ctx?.%s)) || ctx?.%s != ctx?.%s)", currIf, field, field, val, field, val)

	log.Debug("adding if %s to replace 'allow_duplicates: false' in append processor", newIf)
	options["if"] = newIf

	return nil
}
