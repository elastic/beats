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
	"fmt"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-ucfg"
	"github.com/elastic/go-ucfg/parse"

	"github.com/elastic/beats/v7/libbeat/common/bus"
	"github.com/elastic/beats/v7/libbeat/conditions"
	"github.com/elastic/beats/v7/libbeat/keystore"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Mapper maps config templates with conditions in ConditionMaps, if a match happens on a discover event
// the given template will be used as config.
// Mapper also includes the global Keystore object at `keystore` and `keystoreProvider`, which
// has access to a keystores registry
type Mapper struct {
	ConditionMaps    []*ConditionMap
	keystore         keystore.Keystore
	keystoreProvider keystore.Provider
}

// ConditionMap maps a condition to the configs to use when it's triggered
type ConditionMap struct {
	Condition conditions.Condition
	Configs   []*conf.C
}

// MapperSettings holds user settings to build Mapper
type MapperSettings []*struct {
	ConditionConfig *conditions.Config `config:"condition"`
	Configs         []*conf.C          `config:"config"`
}

// NewConfigMapper builds a template Mapper from given settings
func NewConfigMapper(
	configs MapperSettings,
	keystore keystore.Keystore,
	keystoreProvider keystore.Provider,
) (mapper Mapper, err error) {
	for _, c := range configs {
		condMap := &ConditionMap{Configs: c.Configs}
		if c.ConditionConfig != nil {
			condMap.Condition, err = conditions.NewCondition(c.ConditionConfig)
			if err != nil {
				return Mapper{}, err
			}
		}
		mapper.ConditionMaps = append(mapper.ConditionMaps, condMap)
	}

	mapper.keystore = keystore
	mapper.keystoreProvider = keystoreProvider
	return mapper, nil
}

// Event adapts MapStr to processors.ValuesMap interface
type Event mapstr.M

// GetValue extracts given key from an Event
func (e Event) GetValue(key string) (interface{}, error) {
	val, err := mapstr.M(e).GetValue(key)
	if err != nil {
		return nil, err
	}
	return val, nil
}

// GetConfig returns a matching Config if any, nil otherwise
func (c Mapper) GetConfig(event bus.Event) []*conf.C {
	var result []*conf.C
	opts := []ucfg.Option{}
	// add k8s keystore in options list with higher priority
	if c.keystoreProvider != nil {
		k8sKeystore := c.keystoreProvider.GetKeystore(event)
		if k8sKeystore != nil {
			opts = append(opts, ucfg.Resolve(keystore.ResolverWrap(k8sKeystore)))
		}
	}
	// add local keystore in options list with lower priority
	if c.keystore != nil {
		opts = append(opts, ucfg.Resolve(keystore.ResolverWrap(c.keystore)))
	}
	for _, mapping := range c.ConditionMaps {
		// An empty condition matches everything
		conditionOk := mapping.Condition == nil || mapping.Condition.Check(Event(event))
		if mapping.Configs != nil && !conditionOk {
			continue
		}

		configs := ApplyConfigTemplate(event, mapping.Configs, opts...)
		if configs != nil {
			result = append(result, configs...)
		}
	}
	return result
}

// ApplyConfigTemplate takes a set of templated configs and applys information in an event map
func ApplyConfigTemplate(event bus.Event, configs []*conf.C, options ...ucfg.Option) []*conf.C {
	var result []*conf.C
	// unpack input
	vars, err := ucfg.NewFrom(map[string]interface{}{
		"data": event,
	})
	if err != nil {
		logp.Err("Error building config: %v", err)
	}

	opts := []ucfg.Option{
		// Catch-all resolve function to log fields not resolved in any other way,
		// it needs to be the first resolver added, so it is executed the last one.
		// Being the last one, its returned error will be the one returned by `Unpack`,
		// this is important to give better feedback in case of failure.
		ucfg.Resolve(func(name string) (string, parse.Config, error) {
			return "", parse.Config{}, fmt.Errorf("field '%s' not available in event or environment", name)
		}),

		ucfg.PathSep("."),
		ucfg.Env(vars),
		ucfg.ResolveEnv,
		ucfg.VarExp,
	}
	opts = append(opts, options...)

	for _, cfg := range configs {
		c, err := ucfg.NewFrom(cfg, opts...)
		if err != nil {
			logp.Err("Error parsing config: %v", err)
			continue
		}
		// Unpack config to process any vars in the template:
		var unpacked map[string]interface{}
		err = c.Unpack(&unpacked, opts...)
		if err != nil {
			logp.Debug("autodiscover", "Configuration template cannot be resolved: %v", err)
			continue
		}
		// Repack again:
		res, err := conf.NewConfigFrom(unpacked)
		if err != nil {
			logp.Err("Error creating config from unpack: %v", err)
			continue
		}
		result = append(result, res)
	}
	return result
}
