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
	"os"

	"github.com/elastic/beats/libbeat/beat"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type ESClient interface {
	LoadJSON(path string, json map[string]interface{}) ([]byte, error)
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
	GetVersion() common.Version
}

type Loader struct {
	esClient   ESClient
	cfg        ILMConfig
	ilmEnabled bool
}

//NewLoader creates a new ilm policy loader that writes to ES or the console
func NewLoader(cfg ILMConfig, client ESClient, ilmEnabled bool, info beat.Info) (*Loader, error) {
	//ilm.enabled=false
	if cfg.EnabledFalse() {
		return nil, nil
	}

	//setup is not qualified for ILM
	if !ilmEnabled {
		//ilm.enabled=true
		if cfg.EnabledTrue() {
			return nil, errors.New(fmt.Sprintf("ilm set to `true`, but %s", ilmNotSupported))
		}
		//ilm.enabled=auto
		logp.Info(fmt.Sprintf("ilm set to `auto`, but %s", ilmNotSupported))
		return nil, nil
	}

	cfg.replaceRegex(info)
	return &Loader{
		esClient:   client,
		cfg:        cfg,
		ilmEnabled: ilmEnabled,
	}, nil

}

//LoadPolicy loads the configured ILM policy to the appropriate output,
//either Elasticsearch or the Console.
func (l *Loader) LoadPolicy() (bool, error) {
	logp.Info("Starting to load policy %s", l.cfg.Policy.Name)
	policy, err := newPolicy(l.cfg.Policy)
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
func (l *Loader) LoadWriteAlias() (bool, error) {
	logp.Info("Starting to load write alias %s", l.cfg.RolloverAlias)
	exists, err := l.checkAliasExists(l.cfg.RolloverAlias)
	if err != nil {
		logp.Err("Failed to check for alias: %s: ", err)
		return false, err
	}
	if exists {
		logp.Info("Write alias already exists")
		return false, nil
	}

	return l.createAlias(l.cfg.RolloverAlias)
}

func (l *Loader) checkAliasExists(alias string) (bool, error) {
	status, b, err := l.esClient.Request("HEAD", "/_alias/"+alias, "", nil, nil)
	if err != nil && status != 404 {
		return false, errors.New(fmt.Sprintf("%s: %v", err.Error(), string(b)))
	}
	if status == 200 {
		return true, nil
	}
	return false, nil
}

func (l *Loader) createAlias(alias string) (bool, error) {
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
		return errors.New("policy empty")
	}
	if l.esClient != nil {
		if _, _, err := l.esClient.Request("PUT", "/_ilm/policy/"+p.name, "", nil, p.body); err != nil {
			return err
		}
		return nil
	}
	_, err := os.Stdout.WriteString(fmt.Sprintf("%s: %s\n", p.name, p.body))
	return err
}
