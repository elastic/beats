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

package ilm

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	errw "github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

//ESClient supporting methods necessary for ILM handling
type ESClient interface {
	LoadJSON(path string, json map[string]interface{}) ([]byte, error)
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
	GetVersion() common.Version
}

//Loader interface for loading ilm policies and write aliases
type Loader interface {
	LoadPolicy(cfg Config) (bool, error)
	LoadWriteAlias(cfg Config) (bool, error)
}

//ESLoader holds all the information necessary to load ilm policies and write aliases to ES
type ESLoader struct {
	ilmEnabled bool
	esClient   ESClient
	beatInfo   beat.Info
}

//NewESLoader creates a new ilm policy loader for ES
func NewESLoader(client ESClient, info beat.Info) (Loader, error) {
	return &ESLoader{
		esClient:   client,
		ilmEnabled: EnabledFor(client),
		beatInfo:   info,
	}, nil
}

//LoadPolicy loads the configured ILM policy to the configured ES output
func (l *ESLoader) LoadPolicy(cfg Config) (bool, error) {
	logp.Info("Starting to load policy %s", cfg.Policy.Name)

	if load, err := l.shouldLoad(cfg); load == false || err != nil {
		return load, err
	}

	policy, err := preparePolicy(l.beatInfo, cfg)
	if err != nil {
		return false, err
	}

	if l.esClient == nil {
		return false, errors.New("no ES client configured for loading ILM write alias")
	}

	if err := l.policyToES(policy); err != nil {
		logp.Err("Error loading policy: %s", err)
		return false, err
	}

	logp.Info("Policy successfully loaded")
	return true, nil

}

//LoadWriteAlias loads the configured rollover alias to the Elasticsearch output
func (l *ESLoader) LoadWriteAlias(cfg Config) (bool, error) {
	logp.Info("Starting to load write alias %s", cfg.RolloverAlias)

	if load, err := l.shouldLoad(cfg); load == false || err != nil {
		return load, err
	}

	if err := cfg.prepare(l.beatInfo); err != nil {
		return false, err
	}

	if l.esClient == nil {
		return false, errors.New("no ES client configured for loading ILM write alias")
	}

	exists, err := l.checkAliasExists(cfg.RolloverAlias)
	if err != nil {
		logp.Err("Failed to check for alias: %s: ", err)
		return false, err
	}
	if exists {
		logp.Info("Write alias already exists")
		return false, nil
	}

	return l.createAlias(cfg.RolloverAlias, cfg.Pattern)
}

func (l *ESLoader) shouldLoad(cfg Config) (bool, error) {
	//ilm.enabled=false
	if cfg.Enabled == ModeDisabled {
		return false, nil
	}
	if l.ilmEnabled {
		return true, nil
	}

	//setup is not qualified for ILM
	if cfg.Enabled == ModeAuto {
		//ilm.enabled=auto
		logp.Info(fmt.Sprintf("ILM for %s set to `auto`, but %s", cfg.RolloverAlias, ilmNotSupported))
		return false, nil
	}
	//ilm.enabled=true
	return false, fmt.Errorf("ILM set to `true`, but %s", ilmNotSupported)
}

func (l *ESLoader) checkAliasExists(alias string) (bool, error) {
	status, b, err := l.esClient.Request("HEAD", "/_alias/"+alias, "", nil, nil)
	if err != nil && status != 404 {
		return false, fmt.Errorf("%s: %v", err.Error(), string(b))
	}
	if status == 200 {
		return true, nil
	}
	return false, nil
}

func (l *ESLoader) createAlias(alias string, pattern string) (bool, error) {
	firstIndex := fmt.Sprintf("<%s-%s>", alias, pattern)
	//ensure data pattern is properly encoded
	patternParsed, err := url.ParseQuery(firstIndex)
	if err != nil {
		return false, err
	}
	firstIndex = strings.TrimSuffix(patternParsed.Encode(), "=")

	body := common.MapStr{
		"aliases": common.MapStr{
			alias: common.MapStr{
				"is_write_index": true,
			},
		},
	}

	code, res, err := l.esClient.Request("PUT", "/"+firstIndex, "", nil, body)
	if code == 400 {
		if strings.Contains(err.Error(), "already exists") {
			logp.Info("Write alias %s already exists", firstIndex)
			return false, nil
		}
	}
	if err != nil {
		return false, errw.Wrapf(err, string(res))
	}

	logp.Info("Alias with write index successfully created: %s", firstIndex)
	return true, nil
}

func (l *ESLoader) policyToES(p *policy) error {
	if p == nil {
		return fmt.Errorf("policy empty")
	}

	if l.esClient == nil {
		return errors.New("cannot load ILM policy, missing ES client")
	}

	if _, _, err := l.esClient.Request("PUT", "/_ilm/policy/"+p.name, "", nil, p.body); err != nil {
		return err
	}
	return nil
}

//StdoutLoader holds all the information necessary to load ilm policies and write aliases to stdout
type StdoutLoader struct {
	beatInfo beat.Info
}

//NewStdoutLoader creates a new ilm policy loader for stdout
func NewStdoutLoader(info beat.Info) (Loader, error) {
	return &StdoutLoader{beatInfo: info}, nil
}

//LoadPolicy loads the configured ILM policy to stdout
func (l *StdoutLoader) LoadPolicy(cfg Config) (bool, error) {
	logp.Info("Starting to print policy %s", cfg.Policy.Name)
	if cfg.Enabled == ModeDisabled {
		return false, nil
	}

	policy, err := preparePolicy(l.beatInfo, cfg)
	if err != nil {
		return false, err
	}

	//write do stdout
	body, err := policy.String()
	if err != nil {
		return false, err
	}
	if _, err := os.Stdout.WriteString(fmt.Sprintf("Register policy at `/_ilm/policy/%s`\n%v", policy.name, body)); err != nil {
		return false, fmt.Errorf("error writing ILM policy: %v", err)
	}

	return true, nil
}

//LoadWriteAlias does nothing for stdout loader
func (l *StdoutLoader) LoadWriteAlias(cfg Config) (bool, error) {
	//not implemented as not needed yet.
	return false, nil
}

func preparePolicy(beatInfo beat.Info, cfg Config) (*policy, error) {
	if err := cfg.prepare(beatInfo); err != nil {
		return nil, err
	}

	policy, err := newPolicy(cfg.Policy)
	if err != nil {
		logp.Err("Error creating policy: %s", err)
		return nil, err
	}
	return policy, err

}
