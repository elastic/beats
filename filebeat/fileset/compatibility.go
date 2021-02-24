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
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// processorCompatibility defines a processor's minimum version requirements or
// a transformation to make it compatible.
type processorCompatibility struct {
	checkVersion func(esVersion *common.Version) bool                                            // Version check returns true if this check applies.
	procType     string                                                                          // Elasticsearch Ingest Node processor type.
	adaptConfig  func(processor map[string]interface{}, log *logp.Logger) (drop bool, err error) // Adapt the configuration to make it compatible.
}

var processorCompatibilityChecks = []processorCompatibility{
	{
		procType: "append",
		checkVersion: func(esVersion *common.Version) bool {
			return esVersion.LessThan(common.MustNewVersion("7.10.0"))
		},
		adaptConfig: replaceAppendAllowDuplicates,
	},
	{
		procType: "community_id",
		checkVersion: func(esVersion *common.Version) bool {
			return esVersion.LessThan(common.MustNewVersion("7.12.0"))
		},
		adaptConfig: deleteProcessor,
	},
	{
		procType: "set",
		checkVersion: func(esVersion *common.Version) bool {
			return esVersion.LessThan(common.MustNewVersion("7.9.0"))
		},
		adaptConfig: replaceSetIgnoreEmptyValue,
	},
	{
		procType: "uri_parts",
		checkVersion: func(esVersion *common.Version) bool {
			return esVersion.LessThan(common.MustNewVersion("7.12.0"))
		},
		adaptConfig: deleteProcessor,
	},
	{
		procType: "user_agent",
		checkVersion: func(esVersion *common.Version) bool {
			return esVersion.LessThan(common.MustNewVersion("7.0.0")) &&
				!esVersion.LessThan(common.MustNewVersion("6.7.0"))
		},
		adaptConfig: func(config map[string]interface{}, _ *logp.Logger) (bool, error) {
			config["ecs"] = true
			return false, nil
		},
	},
	{
		procType: "user_agent",
		checkVersion: func(esVersion *common.Version) bool {
			return esVersion.LessThan(common.MustNewVersion("6.7.0"))
		},
		adaptConfig: func(config map[string]interface{}, _ *logp.Logger) (bool, error) {
			return false, errors.New("user_agent processor requires option 'ecs: true', Elasticsearch 6.7 or newer required")
		},
	},
}

// adaptPipelineForCompatibility iterates over all processors in the pipeline
// and adapts them for version of Elasticsearch used. Adapt can mean modifying
// processor options or removing the processor.
func adaptPipelineForCompatibility(esVersion common.Version, pipelineID string, content map[string]interface{}, log *logp.Logger) error {
	p, ok := content["processors"]
	if !ok {
		return errors.New("'processors' is missing from the pipeline definition")
	}

	processors, ok := p.([]interface{})
	if !ok {
		return fmt.Errorf("'processors' in pipeline '%s' expected to be a list, found %T", pipelineID, p)
	}

	var filteredProcs []interface{}

nextProcessor:
	for i, obj := range processors {
		processor, ok := obj.(map[string]interface{})
		if !ok {
			return fmt.Errorf("processor at index %d is not an object, got %T", i, obj)
		}

		for _, proc := range processorCompatibilityChecks {
			configIfc, found := processor[proc.procType]
			if !found {
				continue
			}
			config, ok := configIfc.(map[string]interface{})
			if !ok {
				return fmt.Errorf("processor config at index %d is not an object, got %T", i, obj)
			}

			if !proc.checkVersion(&esVersion) {
				continue
			}

			drop, err := proc.adaptConfig(config, log.With("processor_type", proc.procType, "processor_index", i))
			if err != nil {
				return fmt.Errorf("failed to adapt %q processor at index %d: %w", proc.procType, i, err)
			}
			if drop {
				continue nextProcessor
			}
		}

		filteredProcs = append(filteredProcs, processors[i])
	}

	content["processors"] = filteredProcs
	return nil
}

// deleteProcessor returns true to indicate that the processor should be deleted
// in order to adapt the pipeline for backwards compatibility to Elasticsearch.
func deleteProcessor(_ map[string]interface{}, _ *logp.Logger) (bool, error) { return true, nil }

// replaceSetIgnoreEmptyValue replaces ignore_empty_value option with an if
// statement so ES less than 7.9 will work.
func replaceSetIgnoreEmptyValue(config map[string]interface{}, log *logp.Logger) (bool, error) {
	_, ok := config["ignore_empty_value"].(bool)
	if !ok {
		return false, nil
	}

	log.Debug("Removing unsupported 'ignore_empty_value' from set processor.")
	delete(config, "ignore_empty_value")

	_, ok = config["if"].(string)
	if ok {
		// assume if check is sufficient
		return false, nil
	}
	val, ok := config["value"].(string)
	if !ok {
		return false, nil
	}

	newIf := strings.TrimLeft(val, "{ ")
	newIf = strings.TrimRight(newIf, "} ")
	newIf = strings.ReplaceAll(newIf, ".", "?.")
	newIf = "ctx?." + newIf + " != null"

	log.Debug("Adding if %s to replace 'ignore_empty_value' in set processor.", newIf)
	config["if"] = newIf
	return false, nil
}

// replaceAppendAllowDuplicates replaces allow_duplicates option with an if statement
// so ES less than 7.10 will work.
func replaceAppendAllowDuplicates(config map[string]interface{}, log *logp.Logger) (bool, error) {
	allow, ok := config["allow_duplicates"].(bool)
	if !ok {
		return false, nil
	}

	log.Debug("Removing unsupported 'allow_duplicates' from append processor.")
	delete(config, "allow_duplicates")

	if allow {
		// It was set to true, nothing else to do after removing the option.
		return false, nil
	}

	currIf, _ := config["if"].(string)
	if strings.Contains(strings.ToLower(currIf), "contains") {
		// If it has a contains statement, we assume it is checking for duplicates already.
		return false, nil
	}
	field, ok := config["field"].(string)
	if !ok {
		return false, nil
	}
	val, ok := config["value"].(string)
	if !ok {
		return false, nil
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

	log.Debug("Adding if %s to replace 'allow_duplicates: false' in append processor.", newIf)
	config["if"] = newIf

	return false, nil
}
