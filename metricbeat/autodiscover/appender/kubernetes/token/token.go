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

package token

import (
	"fmt"
	"io/ioutil"

	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/conditions"
	"github.com/elastic/elastic-agent-autodiscover/bus"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type tokenAppender struct {
	TokenPath string
	Condition conditions.Condition
}

// InitializeModule initializes this module.
func InitializeModule() {
	err := autodiscover.Registry.AddAppender("kubernetes.token", NewTokenAppender)
	if err != nil {
		logp.Error(fmt.Errorf("could not add `kubernetes.token` appender"))
	}
}

// NewTokenAppender creates a token appender that can append a bearer token required to authenticate with
// protected endpoints
func NewTokenAppender(cfg *conf.C) (autodiscover.Appender, error) {
	cfgwarn.Deprecate("7.0.0", "token appender is deprecated in favor of bearer_token_file config parameter")
	conf := defaultConfig()

	err := cfg.Unpack(&conf)
	if err != nil {
		return nil, fmt.Errorf("unable to unpack config due to error: %v", err)
	}

	var cond conditions.Condition
	if conf.ConditionConfig != nil {
		// Attempt to create a condition. If fails then report error
		cond, err = conditions.NewCondition(conf.ConditionConfig)
		if err != nil {
			return nil, fmt.Errorf("unable to create condition due to error: %v", err)
		}
	}
	appender := tokenAppender{
		TokenPath: conf.TokenPath,
		Condition: cond,
	}

	return &appender, nil
}

// Append picks up a token from a file and adds it to the headers.Authorization section of the metricbeat module
func (t *tokenAppender) Append(event bus.Event) {
	cfgsRaw, ok := event["config"]
	// There are no configs
	if !ok {
		return
	}

	cfgs, ok := cfgsRaw.([]*conf.C)
	// Config key doesnt have an array of config objects
	if !ok {
		return
	}

	// Check if the condition is met. Attempt to append only if that is the case.
	if t.Condition == nil || t.Condition.Check(mapstr.M(event)) == true {
		tok := t.getAuthHeaderFromToken()
		// If token is empty then just return
		if tok == "" {
			return
		}
		for i := 0; i < len(cfgs); i++ {
			// Unpack the config
			cfg := cfgs[i]
			c := mapstr.M{}
			err := cfg.Unpack(&c)
			if err != nil {
				logp.Debug("kubernetes.config", "unable to unpack config due to error: %v", err)
				continue
			}
			var headers mapstr.M
			if hRaw, ok := c["headers"]; ok {
				// If headers is not a map then continue to next config
				if headers, ok = hRaw.(mapstr.M); !ok {
					continue
				}
			} else {
				headers = mapstr.M{}
			}

			// Assign authorization header and add it back to the config
			headers["Authorization"] = tok
			c["headers"] = headers

			// Repack the configuration
			newCfg, err := conf.NewConfigFrom(&c)
			if err != nil {
				logp.Debug("kubernetes.config", "unable to repack config due to error: %v", err)
				continue
			}
			cfgs[i] = newCfg
		}

		event["config"] = cfgs
	}
}

func (t *tokenAppender) getAuthHeaderFromToken() string {
	var token string

	if t.TokenPath != "" {
		b, err := ioutil.ReadFile(t.TokenPath)
		if err != nil {
			logp.Err("Reading token file failed with err: %v", err)
		}

		if len(b) != 0 {
			if b[len(b)-1] == '\n' {
				b = b[0 : len(b)-1]
			}
			token = fmt.Sprintf("Bearer %s", string(b))
		}
	}

	return token
}
