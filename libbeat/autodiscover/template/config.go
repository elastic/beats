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

package template

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/conditions"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/go-ucfg"
)

// Mapper maps config templates with conditions, if a match happens on a discover event
// the given template will be used as config
type Mapper []*ConditionMap

// ConditionMap maps a condition to the configs to use when it's triggered
type ConditionMap struct {
	Condition conditions.Condition
	Configs   []*common.Config
}

// MapperSettings holds user settings to build Mapper
type MapperSettings []*struct {
	ConditionConfig *conditions.Config `config:"condition"`
	Configs         []*common.Config   `config:"config"`
}

// NewConfigMapper builds a template Mapper from given settings
func NewConfigMapper(configs MapperSettings) (mapper Mapper, err error) {
	for _, c := range configs {
		condMap := &ConditionMap{Configs: c.Configs}
		if c.ConditionConfig != nil {
			condMap.Condition, err = conditions.NewCondition(c.ConditionConfig)
			if err != nil {
				return nil, err
			}
		}

		mapper = append(mapper, condMap)
	}

	return mapper, nil
}

// Event adapts MapStr to processors.ValuesMap interface
type Event common.MapStr

// GetValue extracts given key from an Event
func (e Event) GetValue(key string) (interface{}, error) {
	val, err := common.MapStr(e).GetValue(key)
	if err != nil {
		return nil, err
	}
	return val, nil
}

// GetConfig returns a matching Config if any, nil otherwise
func (c Mapper) GetConfig(event bus.Event) []*common.Config {
	var result []*common.Config

	for _, mapping := range c {
		// An empty condition matches everything
		conditionOk := mapping.Condition == nil || mapping.Condition.Check(Event(event))
		if mapping.Configs != nil && !conditionOk {
			continue
		}

		configs := ApplyConfigTemplate(event, mapping.Configs)
		if configs != nil {
			result = append(result, configs...)
		}
	}
	return result
}

// ApplyConfigTemplate takes a set of templated configs and applys information in an event map
func ApplyConfigTemplate(event bus.Event, configs []*common.Config) []*common.Config {
	var result []*common.Config
	// unpack input
	vars, err := ucfg.NewFrom(map[string]interface{}{
		"data": event,
	})
	if err != nil {
		logp.Err("Error building config: %v", err)
	}
	opts := []ucfg.Option{
		ucfg.PathSep("."),
		ucfg.Env(vars),
		ucfg.ResolveEnv,
		ucfg.VarExp,
	}
	for _, config := range configs {
		c, err := ucfg.NewFrom(config, opts...)
		if err != nil {
			logp.Err("Error parsing config: %v", err)
			continue
		}
		// Unpack config to process any vars in the template:
		var unpacked map[string]interface{}
		c.Unpack(&unpacked, opts...)
		if err != nil {
			logp.Err("Error unpacking config: %v", err)
			continue
		}
		// Repack again:
		res, err := common.NewConfigFrom(unpacked)
		if err != nil {
			logp.Err("Error creating config from unpack: %v", err)
			continue
		}
		result = append(result, res)
	}
	return result
}
