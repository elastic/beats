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
	"encoding/json"
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/template"
)

type ESClient interface {
	LoadJSON(path string, json map[string]interface{}) ([]byte, error)
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
	GetVersion() common.Version
}

type Loader struct {
	esClient         ESClient
	beatInfo         beat.Info
	ilmPolicyConfigs []ilmPolicyCfg
}

// NewESLoader creates a new ilm policy loader that writes to ES
func NewESLoader(outCfg *common.Config, client ESClient, beatInfo beat.Info) (*Loader, error) {
	return newLoader(outCfg, client, beatInfo)
}

// NewConsoleLoader creates a new ilm policy loader that writes to the Cosole
func NewConsoleLoader(outCfg *common.Config, beatInfo beat.Info) (*Loader, error) {
	return newLoader(outCfg, nil, beatInfo)
}

func newLoader(outCfg *common.Config, esClient ESClient, beatInfo beat.Info) (*Loader, error) {
	// Get ES Index name for comparison
	esCfg := struct {
		Index          string                   `config:"index"`
		RolloverPolicy string                   `config:"rollover_policy"`
		Indices        []map[string]interface{} `config:"indices"`
	}{}
	err := outCfg.Unpack(&esCfg)
	if err != nil {
		return nil, err
	}

	//find indices with rollover policy definitions
	var ilmPolicyConfigs []ilmPolicyCfg
	if esCfg.Index != "" && esCfg.RolloverPolicy != "" {
		if cfg := newIlmPolicyCfg(esCfg.Index, esCfg.RolloverPolicy, beatInfo); cfg != nil {
			ilmPolicyConfigs = append(ilmPolicyConfigs, *cfg)
		}
	}
	if esCfg.Indices != nil {
		for _, idxCfg := range esCfg.Indices {
			policy, ok := idxCfg["rollover_policy"]
			if !ok {
				continue
			}

			p, policyOk := policy.(string)
			idx, idxOk := idxCfg["index"].(string)
			if !idxOk || !policyOk {
				continue
			}

			if cfg := newIlmPolicyCfg(idx, p, beatInfo); cfg != nil {
				ilmPolicyConfigs = append(ilmPolicyConfigs, *cfg)
			}
		}
	}

	return &Loader{
		esClient:         esClient,
		ilmPolicyConfigs: ilmPolicyConfigs,
		beatInfo:         beatInfo,
	}, nil
}

func (l *Loader) LoadPolicies() error {
	if l.esClient != nil {
		if err := l.checkILMPreconditions(); err != nil {
			return err
		}
	}

	for _, policyCfg := range l.ilmPolicyConfigs {
		if err := l.loadPolicy(policyCfg); err != nil {
			logp.Err("error loading policy %s: %v", policyCfg.policyName, err)
		}
	}
	return nil
}

func (l *Loader) LoadWriteAlias() error {
	if err := l.checkILMPreconditions(); err != nil {
		return err
	}

	for _, policyCfg := range l.ilmPolicyConfigs {
		rolloverAlias := policyCfg.idxName

		if exists, err := l.checkAliasExists(rolloverAlias); err != nil || exists {
			continue
		}

		if created, err := l.createAlias(rolloverAlias); err != nil || !created {
			continue
		}

		l.createILMTemplate(policyCfg)
	}

	return nil
}

func (l *Loader) checkAliasExists(alias string) (bool, error) {
	status, b, err := l.esClient.Request("HEAD", "/_alias/"+alias, "", nil, nil)
	if err != nil && status != 404 {
		logp.Err("Failed to check for alias: %s: %+v", err, string(b))
		return false, err
	}
	if status == 200 {
		logp.Info("Write alias already exists")
		return true, nil
	}
	return false, nil
}

func (l *Loader) createAlias(alias string) (bool, error) {
	firstIndex := fmt.Sprintf("%s-%s", alias, pattern)
	body := common.MapStr{
		"aliases": common.MapStr{
			alias: common.MapStr{
				"is_write_index": true,
			},
		},
	}

	code, res, err := l.esClient.Request("PUT", "/"+firstIndex, "", nil, body)
	if code == 400 {
		logp.Err("Error creating alias with write index. As return code is 400, assuming already exists: %s, %s", err, string(res))
		return false, nil
	} else if err != nil {
		logp.Err("Error creating alias with write index: %s, %s", err, string(res))
		return false, err
	}

	logp.Info("Alias with write index created: %s", firstIndex)
	return true, nil
}

func (l *Loader) createILMTemplate(policyCfg ilmPolicyCfg) (bool, error) {
	ilmTemplate := policyCfg.buildILMTemplate()
	templateLoader, err := template.NewESLoader(common.NewConfig(), l.esClient, l.beatInfo)
	if err != nil {
		logp.Err("Error creating ILM template loader for index: %s, %s", policyCfg.idxName, err)
		return false, err
	}
	templateName := fmt.Sprintf("%s-ilm", policyCfg.idxName)
	if err := templateLoader.LoadTemplate(templateName, ilmTemplate); err != nil {
		logp.Err("Error loading ILM template for index: %s, %s", policyCfg.idxName, err)
		return false, err
	}
	logp.Info("ilm template %s created", templateName)
	return true, nil
}

func (l *Loader) loadPolicy(policyCfg ilmPolicyCfg) error {
	policy, ok := policies[policyCfg.policyName]
	if !ok {
		logp.Warn("cannot find policy %s", policyCfg.policyName)
		return nil
	}

	if l.esClient != nil {
		_, _, err := l.esClient.Request("PUT", "/_ilm/policy/"+policyCfg.policyName, "", nil, policy)
		return err
	}
	_, err := os.Stdout.WriteString(fmt.Sprintf("%s: %s\n", policyCfg.policyName, policy))
	return err
}

func (l *Loader) checkILMPreconditions() error {
	if len(l.ilmPolicyConfigs) == 0 {
		return nil
	}
	logp.Warn("Index Life Cycle Management (ILM) is in beta!")

	if err := l.checkElasticsearchVersionIlm(); err != nil {
		return err
	}
	return l.checkILMFeatureEnabled()
}

func (l *Loader) checkElasticsearchVersionIlm() error {
	if l.esClient == nil {
		return errors.New("no Elasticsearch client is set")
	}
	esV := l.esClient.GetVersion()
	requiredVersion, err := common.NewVersion("6.6.0")
	if err != nil {
		return err
	}

	if esV.LessThan(requiredVersion) {
		return fmt.Errorf("ILM requires at least Elasticsearch 6.6.0. Used version: %s", esV.String())
	}

	return nil
}

func (l *Loader) checkILMFeatureEnabled() error {
	if l.esClient == nil {
		return errors.New("no Elasticsearch client is set")
	}

	code, body, err := l.esClient.Request("GET", "/_xpack", "", nil, nil)

	// If we get a 400, it's assumed to be the OSS version of Elasticsearch
	if code == 400 {
		return fmt.Errorf("ILM feature is not available in this Elasticsearch version")
	}
	if err != nil {
		return err
	}

	var response struct {
		Features struct {
			ILM struct {
				Available bool `json:"available"`
				Enabled   bool `json:"enabled"`
			} `json:"ilm"`
		} `json:"features"`
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return fmt.Errorf("failed to parse JSON response: %v", err)
	}

	if !response.Features.ILM.Available {
		return fmt.Errorf("ILM feature is not available in Elasticsearch")
	}

	if !response.Features.ILM.Enabled {
		return fmt.Errorf("ILM feature is not enabled in Elasticsearch")
	}

	return nil
}
