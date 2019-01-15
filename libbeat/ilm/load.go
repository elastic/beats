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
	"fmt"
	"os"

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

//Loader holds all the information necessary to load ilm policies and write aliases
type Loader struct {
	ilmEnabled bool
	esClient   ESClient
	beatInfo   beat.Info
}

//NewESLoader creates a new ilm policy loader that writes to ES
func NewESLoader(client ESClient, info beat.Info) (*Loader, error) {
	return &Loader{
		esClient:   client,
		ilmEnabled: EnabledFor(client),
		beatInfo:   info,
	}, nil
}

//NewStdoutLoader creates a new ilm policy loader that writes to the console
func NewStdoutLoader(info beat.Info) (*Loader, error) {
	return &Loader{
		ilmEnabled: true,
		beatInfo:   info,
	}, nil
}

//LoadPolicy loads the configured ILM policy to the appropriate output,
//either Elasticsearch or the Console.
func (l *Loader) LoadPolicy(cfg Config) (bool, error) {
	logp.Info("Starting to load policy %s", cfg.Policy.Name)

	if load, err := l.shouldLoad(cfg); load == false || err != nil {
		return load, err
	}

	if err := cfg.prepare(l.beatInfo); err != nil {
		return false, err
	}

	policy, err := newPolicy(cfg.Policy)
	if err != nil {
		logp.Err("Error creating policy: %s", err)
		return false, err
	}

	if err := l.loadPolicy(policy); err != nil {
		logp.Err("Error loading policy: %s", err)
		return false, err
	}

	logp.Info("Policy successfully loaded")
	return true, nil
}

//LoadWriteAlias loads the configured rollover alias to the Elasticsearch output
func (l *Loader) LoadWriteAlias(cfg Config) (bool, error) {
	logp.Info("Starting to load write alias %s", cfg.RolloverAlias)

	if load, err := l.shouldLoad(cfg); load == false || err != nil {
		return load, err
	}

	if err := cfg.prepare(l.beatInfo); err != nil {
		return false, err
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

	return l.createAlias(cfg.RolloverAlias)
}

func (l *Loader) shouldLoad(cfg Config) (bool, error) {
	//ilm.enabled=false
	if cfg.EnabledFalse() {
		return false, nil
	}
	if l.ilmEnabled {
		return true, nil
	}

	//setup is not qualified for ILM
	if cfg.EnabledAuto() {
		//ilm.enabled=auto
		logp.Info(fmt.Sprintf("ilm for %s set to `auto`, but %s", cfg.RolloverAlias, ilmNotSupported))
		return false, nil
	}
	//ilm.enabled=true
	return false, fmt.Errorf("ilm set to `true`, but %s", ilmNotSupported)
}

func (l *Loader) checkAliasExists(alias string) (bool, error) {
	status, b, err := l.esClient.Request("HEAD", "/_alias/"+alias, "", nil, nil)
	if err != nil && status != 404 {
		return false, fmt.Errorf("%s: %v", err.Error(), string(b))
	}
	if status == 200 {
		return true, nil
	}
	return false, nil
}

func (l *Loader) createAlias(alias string) (bool, error) {
	if l.esClient == nil {
		logp.Info("No ES client configured for loading ILM write alias")
		return false, nil
	}
	firstIndex := fmt.Sprintf("%s-%s", alias, defaultPattern)
	body := common.MapStr{
		"aliases": common.MapStr{
			alias: common.MapStr{
				"is_write_index": true,
			},
		},
	}

	code, res, err := l.esClient.Request("PUT", "/"+firstIndex, "", nil, body)
	if code == 400 {
		logp.Info("Error creating alias with write index. As return code is 400, assuming already exists: %s, %s", err, string(res))
		return false, nil
	} else if err != nil {
		return false, err
	}

	logp.Info("Alias with write index successfully created: %s", firstIndex)
	return true, nil
}

func (l *Loader) loadPolicy(p *policy) error {
	if p == nil {
		return fmt.Errorf("policy empty")
	}
	if l.esClient != nil {
		if _, _, err := l.esClient.Request("PUT", "/_ilm/policy/"+p.name, "", nil, p.body); err != nil {
			return err
		}
		return nil
	}
	if _, err := os.Stdout.WriteString(fmt.Sprintf("Register policy at `/_ilm/policy/%s`\n%v", p.name, p.body)); err != nil {
		return fmt.Errorf("error writing ilm policy: %v", err)
	}

	_, err := os.Stdout.WriteString(fmt.Sprintf("%s: %s\n", p.name, p.body))
	return err
}
