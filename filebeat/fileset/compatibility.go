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

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// processorCompatibility defines a processor's minimum version requirements or
// a transformation to make it compatible.
type processorCompatibility struct {
	checkVersion func(esVersion *common.Version) bool                           // Version check returns true if this check applies.
	procType     string                                                         // Elasticsearch Ingest Node processor type.
	adaptConfig  func(processor Processor, log *logp.Logger) (Processor, error) // Adapt the configuration to make it compatible.
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
		adaptConfig: func(processor Processor, _ *logp.Logger) (Processor, error) {
			processor.Set("ecs", true)
			return processor, nil
		},
	},
	{
		procType: "user_agent",
		checkVersion: func(esVersion *common.Version) bool {
			return esVersion.LessThan(common.MustNewVersion("6.7.0"))
		},
		adaptConfig: func(_ Processor, _ *logp.Logger) (Processor, error) {
			return Processor{}, errors.New("user_agent processor requires option 'ecs: true', Elasticsearch 6.7 or newer required")
		},
	},
	{
		procType: "convert",
		checkVersion: func(esVersion *common.Version) bool {
			return esVersion.LessThan(common.MustNewVersion("7.13.0"))
		},
		adaptConfig: replaceConvertIP,
	},
	{
		procType: "network_direction",
		checkVersion: func(esVersion *common.Version) bool {
			return esVersion.LessThan(common.MustNewVersion("7.13.0"))
		},
		adaptConfig: deleteProcessor,
	},
	{
		procType: "registered_domain",
		checkVersion: func(esVersion *common.Version) bool {
			return esVersion.LessThan(common.MustNewVersion("7.13.0"))
		},
		adaptConfig: deleteProcessor,
	},
	{
		procType: "*",
		checkVersion: func(esVersion *common.Version) bool {
			return esVersion.LessThan(common.MustNewVersion("7.9.0"))
		},
		adaptConfig: removeDescription,
	},
}

// Processor represents and Ingest Node processor definition.
type Processor struct {
	name   string
	config map[string]interface{}
}

// NewProcessor returns the representation of an Ingest Node processor
// for the given configuration.
func NewProcessor(raw interface{}) (p Processor, err error) {
	rawAsMap, ok := raw.(map[string]interface{})
	if !ok {
		return p, fmt.Errorf("processor is not an object, got %T", raw)
	}

	var keys []string
	for k := range rawAsMap {
		keys = append(keys, k)
	}
	if len(keys) != 1 {
		return p, fmt.Errorf("processor doesn't have exactly 1 key, got %d: %v", len(keys), keys)
	}
	p.name = keys[0]
	if p.config, ok = rawAsMap[p.name].(map[string]interface{}); !ok {
		return p, fmt.Errorf("processor config is not an object, got %T", rawAsMap[p.name])
	}
	return p, nil
}

// Name of the processor.
func (p *Processor) Name() string {
	return p.name
}

// IsNil returns a boolean indicating if the processor is the zero value.
func (p *Processor) IsNil() bool {
	return p.name == ""
}

// Config returns the processor configuration as a map.
func (p *Processor) Config() map[string]interface{} {
	return p.config
}

// GetBool returns a boolean flag from the processor's configuration.
func (p *Processor) GetBool(key string) (value, ok bool) {
	value, ok = p.config[key].(bool)
	return
}

// GetString returns a string flag from the processor's configuration.
func (p *Processor) GetString(key string) (value string, ok bool) {
	value, ok = p.config[key].(string)
	return
}

// GetList returns an array from the processor's configuration.
func (p *Processor) GetList(key string) (value []interface{}, ok bool) {
	value, ok = p.config[key].([]interface{})
	return
}

// Set a flag in the processor's configuration.
func (p *Processor) Set(key string, value interface{}) {
	p.config[key] = value
}

// Get a flag from the processor's configuration.
func (p *Processor) Get(key string) (value interface{}, ok bool) {
	value, ok = p.config[key]
	return
}

// Delete a configuration flag.
func (p *Processor) Delete(key string) {
	delete(p.config, key)
}

// ToMap returns the representation for the processor as a map.
func (p *Processor) ToMap() map[string]interface{} {
	return map[string]interface{}{
		p.name: p.config,
	}
}

// String returns a string representation for the processor.
func (p *Processor) String() string {
	b, err := json.Marshal(p.ToMap())
	if err != nil {
		return fmt.Sprintf("/* encoding error: %v */", err)
	}
	return string(b)
}

// adaptPipelineForCompatibility iterates over all processors in the pipeline
// and adapts them for version of Elasticsearch used. Adapt can mean modifying
// processor options or removing the processor.
func AdaptPipelineForCompatibility(esVersion common.Version, pipelineID string, content map[string]interface{}, log *logp.Logger) (err error) {
	log = log.With("pipeline_id", pipelineID)
	// Adapt the main processors in the pipeline.
	if err = adaptProcessorsForCompatibility(esVersion, content, "processors", false, log); err != nil {
		return err
	}
	// Adapt any `on_failure` processors in the pipeline.
	return adaptProcessorsForCompatibility(esVersion, content, "on_failure", true, log)
}

func adaptProcessorsForCompatibility(esVersion common.Version, content map[string]interface{}, section string, ignoreMissingsection bool, log *logp.Logger) (err error) {
	p, ok := content[section]
	if !ok {
		if ignoreMissingsection {
			return nil
		}
		return fmt.Errorf("'%s' is missing from the pipeline definition", section)
	}

	processors, ok := p.([]interface{})
	if !ok {
		return fmt.Errorf("'%s' expected to be a list, found %T", section, p)
	}

	var filteredProcs []interface{}
	log = log.With("processors_section", section)

nextProcessor:
	for i, obj := range processors {
		processor, err := NewProcessor(obj)
		if err != nil {
			return errors.Wrapf(err, "cannot parse processor in section '%s' index %d body=%+v", section, i, obj)
		}

		// Adapt any on_failure processors for this processor.
		prevOnFailure, _ := processor.GetList("on_failure")
		if err = adaptProcessorsForCompatibility(esVersion, processor.Config(), "on_failure", true,
			log.With("parent_processor_type", processor.Name(), "parent_processor_index", i)); err != nil {
			return errors.Wrapf(err, "cannot parse on_failure for processor in section '%s' index %d body=%+v", section, i, obj)
		}
		if onFailure, _ := processor.GetList("on_failure"); len(prevOnFailure) > 0 && len(onFailure) == 0 {
			processor.Delete("on_failure")
		}

		// Adapt inner processor in case of foreach.
		if inner, found := processor.Get("processor"); found && processor.Name() == "foreach" {
			processor.Set("processor", []interface{}{inner})
			if err = adaptProcessorsForCompatibility(esVersion, processor.Config(), "processor", false,
				log.With("parent_processor_type", processor.Name(), "parent_processor_index", i)); err != nil {
				return errors.Wrapf(err, "cannot parse inner processor for foreach in section '%s' index %d", section, i)
			}
			newList, _ := processor.GetList("processor")
			switch len(newList) {
			case 0:
				// compatibility has removed the inner processor of a foreach processor,
				// must also remove the foreach processor itself.
				continue nextProcessor
			case 1:
				// replace existing processor with possibly modified one.
				processor.Set("processor", newList[0])
			default:
				// This is actually not possible as compatibility checks
				// can't inject extra processors.
				return fmt.Errorf("parsing inner processor for foreach in section '%s' index %d results in more than one processor, which is unsupported by foreach", section, i)
			}
		}

		// Run compatibility checks on the processor.
		for _, proc := range processorCompatibilityChecks {
			if processor.Name() != proc.procType && proc.procType != "*" {
				continue
			}

			if !proc.checkVersion(&esVersion) {
				continue
			}

			processor, err = proc.adaptConfig(processor, log.With("processor_type", processor.Name(), "processor_index", i))
			if err != nil {
				return fmt.Errorf("failed to adapt %q processor at index %d: %w", processor.Name(), i, err)
			}
			if processor.IsNil() {
				continue nextProcessor
			}
		}

		filteredProcs = append(filteredProcs, processor.ToMap())
	}

	content[section] = filteredProcs
	return nil
}

// deleteProcessor returns true to indicate that the processor should be deleted
// in order to adapt the pipeline for backwards compatibility to Elasticsearch.
func deleteProcessor(_ Processor, _ *logp.Logger) (Processor, error) {
	return Processor{}, nil
}

// replaceSetIgnoreEmptyValue replaces ignore_empty_value option with an if
// statement so ES less than 7.9 will work.
func replaceSetIgnoreEmptyValue(processor Processor, log *logp.Logger) (Processor, error) {
	_, ok := processor.GetBool("ignore_empty_value")
	if !ok {
		return processor, nil
	}

	log.Debug("Removing unsupported 'ignore_empty_value' from set processor.")
	processor.Delete("ignore_empty_value")

	_, ok = processor.GetString("if")
	if ok {
		// assume if check is sufficient
		return processor, nil
	}
	val, ok := processor.GetString("value")
	if !ok {
		return processor, nil
	}

	newIf := strings.TrimLeft(val, "{ ")
	newIf = strings.TrimRight(newIf, "} ")
	newIf = strings.ReplaceAll(newIf, ".", "?.")
	newIf = "ctx?." + newIf + " != null"

	log.Debugf("Adding if %s to replace 'ignore_empty_value' in set processor.", newIf)
	processor.Set("if", newIf)
	return processor, nil
}

// replaceAppendAllowDuplicates replaces allow_duplicates option with an if statement
// so ES less than 7.10 will work.
func replaceAppendAllowDuplicates(processor Processor, log *logp.Logger) (Processor, error) {
	allow, ok := processor.GetBool("allow_duplicates")
	if !ok {
		return processor, nil
	}

	log.Debug("Removing unsupported 'allow_duplicates' from append processor.")
	processor.Delete("allow_duplicates")

	if allow {
		// It was set to true, nothing else to do after removing the option.
		return processor, nil
	}

	currIf, _ := processor.GetString("if")
	if strings.Contains(strings.ToLower(currIf), "contains") {
		// If it has a contains statement, we assume it is checking for duplicates already.
		return processor, nil
	}
	field, ok := processor.GetString("field")
	if !ok {
		return processor, nil
	}
	val, ok := processor.GetString("value")
	if !ok {
		return processor, nil
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

	log.Debugf("Adding if %s to replace 'allow_duplicates: false' in append processor.", newIf)
	processor.Set("if", newIf)

	return processor, nil
}

// replaceConvertIP replaces convert processors with type: ip with a grok expression that uses
// the IP pattern.
func replaceConvertIP(processor Processor, log *logp.Logger) (Processor, error) {
	if wantedType, _ := processor.GetString("type"); wantedType != "ip" {
		return processor, nil
	}
	log.Debug("processor input=", processor.String())
	processor.Delete("type")
	var srcIf, dstIf interface{}
	var found bool
	if srcIf, found = processor.Get("field"); !found {
		return Processor{}, errors.New("field option is required for convert processor")
	}
	if dstIf, found = processor.Get("target_field"); found {
		processor.Delete("target_field")
	} else {
		dstIf = srcIf
	}
	processor.Set("patterns", []string{
		fmt.Sprintf("^%%{IP:%s}$", dstIf),
	})
	processor.name = "grok"
	log.Debug("processor output=", processor.String())
	return processor, nil
}

// removeDescription removes the description config option so ES less than 7.9 will work.
func removeDescription(processor Processor, log *logp.Logger) (Processor, error) {
	_, ok := processor.GetString("description")
	if !ok {
		return processor, nil
	}

	log.Debug("Removing unsupported 'description' from processor.")
	processor.Delete("description")

	return processor, nil
}
