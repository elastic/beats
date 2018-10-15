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

package config

import (
	"fmt"

	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/conditions"
	"github.com/elastic/beats/libbeat/logp"
)

func init() {
	autodiscover.Registry.AddAppender("config", NewConfigAppender)
}

type config struct {
	ConditionConfig *conditions.Config `config:"condition"`
	Config          *common.Config     `config:"config"`
}

type configs []config

type configMap struct {
	condition conditions.Condition
	config    common.MapStr
}

type configAppender struct {
	configMaps []configMap
}

// NewConfigAppender creates a configAppender that can append templatized configs into built configs
func NewConfigAppender(cfg *common.Config) (autodiscover.Appender, error) {
	cfgwarn.Beta("The config appender is beta")

	confs := configs{}
	err := cfg.Unpack(&confs)
	if err != nil {
		return nil, fmt.Errorf("unable to unpack config appender due to error: %v", err)
	}

	var configMaps []configMap
	for _, conf := range confs {
		var cond conditions.Condition

		if conf.ConditionConfig != nil {
			cond, err = conditions.NewCondition(conf.ConditionConfig)
			if err != nil {
				logp.Warn("config", "unable to create condition due to error: %v", err)
				continue
			}
		}
		cm := configMap{condition: cond}

		// Unpack the config
		cf := common.MapStr{}
		err = conf.Config.Unpack(&cf)
		if err != nil {
			logp.Warn("config", "unable to unpack config due to error: %v", err)
			continue
		}
		cm.config = cf
		configMaps = append(configMaps, cm)
	}
	return &configAppender{configMaps: configMaps}, nil
}

// Append adds configuration into configs built by builds/templates. It applies conditions to filter out
// configs to apply, applies them and tries to apply templates if any are present.
func (c *configAppender) Append(event bus.Event) {
	cfgsRaw, ok := event["config"]
	// There are no configs
	if !ok {
		return
	}

	cfgs, ok := cfgsRaw.([]*common.Config)
	// Config key doesnt have an array of config objects
	if !ok {
		return
	}
	for _, configMap := range c.configMaps {
		if configMap.condition == nil || configMap.condition.Check(common.MapStr(event)) == true {
			// Merge the template with all the configs
			for _, cfg := range cfgs {
				cf := common.MapStr{}
				err := cfg.Unpack(&cf)
				if err != nil {
					logp.Debug("config", "unable to unpack config due to error: %v", err)
					continue
				}
				err = cfg.Merge(&configMap.config)
				if err != nil {
					logp.Debug("config", "unable to merge configs due to error: %v", err)
				}
			}

			// Apply the template
			template.ApplyConfigTemplate(event, cfgs)
		}
	}

	// Replace old config with newly appended configs
	event["config"] = cfgs
}
