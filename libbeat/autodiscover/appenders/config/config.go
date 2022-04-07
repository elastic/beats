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

	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/autodiscover"
	"github.com/elastic/beats/v8/libbeat/autodiscover/template"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/common/bus"
	"github.com/elastic/beats/v8/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v8/libbeat/conditions"
	"github.com/elastic/beats/v8/libbeat/logp"
)

func init() {
	autodiscover.Registry.AddAppender("config", NewConfigAppender)
}

type config struct {
	ConditionConfig *conditions.Config `config:"condition"`
	Config          *common.Config     `config:"config"`
}

type configAppender struct {
	condition conditions.Condition
	config    common.MapStr
}

// NewConfigAppender creates a configAppender that can append templatized configs into built configs
func NewConfigAppender(cfg *common.Config) (autodiscover.Appender, error) {
	cfgwarn.Beta("The config appender is beta")

	config := config{}
	err := cfg.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("unable to unpack config appender due to error: %+v", err)
	}

	var cond conditions.Condition

	if config.ConditionConfig != nil {
		cond, err = conditions.NewCondition(config.ConditionConfig)
		if err != nil {
			return nil, errors.Wrap(err, "unable to create condition due to error")
		}
	}

	// Unpack the config
	cf := common.MapStr{}
	err = config.Config.Unpack(&cf)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unpack config due to error")
	}

	return &configAppender{condition: cond, config: cf}, nil
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
	if c.condition == nil || c.condition.Check(common.MapStr(event)) == true {
		// Merge the template with all the configs
		for _, cfg := range cfgs {
			cf := common.MapStr{}
			err := cfg.Unpack(&cf)
			if err != nil {
				logp.Debug("config", "unable to unpack config due to error: %v", err)
				continue
			}
			err = cfg.Merge(&c.config)
			if err != nil {
				logp.Debug("config", "unable to merge configs due to error: %v", err)
			}
		}

		// Apply the template
		template.ApplyConfigTemplate(event, cfgs)
	}

	// Replace old config with newly appended configs
	event["config"] = cfgs
}
