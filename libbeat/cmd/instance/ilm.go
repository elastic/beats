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

package instance

import (
	"fmt"

	"github.com/pkg/errors"

	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
)

type ilmConfig struct {
	RolloverAlias string `config:"ilm.rollover_alias" `
	Pattern       string `config:"ilm.pattern"`
}

var ILMPolicy = common.MapStr{
	"policy": common.MapStr{
		"type": "timeseries",
		"phases": common.MapStr{
			"hot": common.MapStr{
				"actions": common.MapStr{
					"rollover": common.MapStr{
						"max_size": "50gb",
						"max_age":  "30d",
					},
				},
			},
		},
	},
}

const (
	ILMPolicyName     = "beats-default-policy"
	ILMDefaultPattern = "00001"
)

// Build and return a callback to load ILM write alias
func (b *Beat) writeAliasLoadingCallback() (func(esClient *elasticsearch.Client) error, error) {
	callback := func(esClient *elasticsearch.Client) error {
		if b.Config.ILM == nil {
			b.Config.ILM = common.NewConfig()
		}

		config, err := getILMConfig(b)
		if err != nil {
			return err
		}

		firstIndex := fmt.Sprintf("%s-%s", config.RolloverAlias, config.Pattern)

		// Check if alias already exists
		status, b, err := esClient.Request("HEAD", "/_alias/"+config.RolloverAlias, "", nil, nil)
		if err != nil && status != 404 {
			logp.Err("Failed create write alias: %s: %+v", err, string(b))
			return errors.Wrap(err, "failed to check for write alias")
		}
		if status == 200 {
			logp.Info("Write alias already exists")
			return nil
		}

		body := common.MapStr{
			"aliases": common.MapStr{
				config.RolloverAlias: common.MapStr{
					"is_write_index": true,
				},
			},
		}

		// Create write alias
		_, res, err := esClient.Request("PUT", "/"+firstIndex, "", nil, body)
		if err != nil {
			logp.Err("Error alias creating stuff: %s, %s", err, string(res))
			return errors.Wrap(err, "failed to create write alias: "+string(res))
		}

		logp.Info("Write alias created")

		return nil
	}

	return callback, nil
}

func (b *Beat) loadILMPolicy() error {

	esClient, err := getElasticsearchClient(b)
	if err != nil {
		return err
	}

	_, _, err = esClient.Request("PUT", "/_ilm/"+ILMPolicyName, "", nil, ILMPolicy)
	return err
}

func getElasticsearchClient(b *Beat) (*elasticsearch.Client, error) {
	outCfg := b.Config.Output
	if outCfg.Name() != "elasticsearch" {
		return nil, fmt.Errorf("Policy loading requested but the Elasticsearch output is not configured/enabled")
	}

	esConfig := outCfg.Config()

	return elasticsearch.NewConnectedClient(esConfig)
}

func loadConfigWithDefaults(config *ilmConfig, b *Beat) {
	if config.RolloverAlias == "" {
		config.RolloverAlias = fmt.Sprintf("%s-%s", b.Info.Beat, b.Info.Version)
	}

	if config.Pattern == "" {
		config.Pattern = ILMDefaultPattern
	}
}

func checkElasticsearchVersionIlm(client *elasticsearch.Client) error {
	esV := client.GetVersion()
	esVersion, err := common.NewVersion(esV)
	if err != nil {
		return err
	}

	requiredVersion, err := common.NewVersion("6.5.0")
	if err != nil {
		return err
	}

	if esVersion.LessThan(requiredVersion) {
		return fmt.Errorf("ILM requires at least Elasticsearc 6.5.0. Used version: %s", esV)
	}

	return nil
}

func checkILMFeatureEnabled(client *elasticsearch.Client) error {
	code, body, err := client.Request("GET", "/_xpack", "", nil, nil)

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

func getILMConfig(b *Beat) (*ilmConfig, error) {
	config := &ilmConfig{}
	err := b.Config.Output.Config().Unpack(config)
	if err != nil {
		return nil, errors.Wrap(err, "problem unpacking ilm configs")
	}

	loadConfigWithDefaults(config, b)

	return config, nil
}
