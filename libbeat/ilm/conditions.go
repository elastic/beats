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

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

//EnabledFor checks if the given Elasticsearch client is valid for ILM.
func EnabledFor(client ESClient) bool {
	if client == nil {
		return false
	}
	if !checkElasticsearchVersionIlm(client) {
		return false
	}
	return checkILMFeatureEnabled(client)
}

func checkElasticsearchVersionIlm(esClient ESClient) bool {
	if esClient == nil {
		logp.Warn(noElasticsearchClientSet)
		return false
	}
	esV := esClient.GetVersion()
	if !esV.IsValid() {
		logp.Warn("unknown Elasticsearch version")
		return false
	}
	requiredVersion, err := common.NewVersion("6.6.0")
	if err != nil {
		logp.Error(err)
		return false
	}

	if esV.LessThan(requiredVersion) {
		logp.Warn(ilmNotSupported)
		return false
	}

	return true
}

func checkILMFeatureEnabled(client ESClient) bool {
	if client == nil {
		logp.Warn(noElasticsearchClientSet)
		return false
	}

	code, body, err := client.Request("GET", "/_xpack", "", nil, nil)
	// If we get a 400, it's assumed to be the OSS version of Elasticsearch
	if code == 400 {
		logp.Warn(ilmNotSupported)
		return false
	}
	if err != nil {
		logp.Err("error occured when checking for ILM features in Elasticsearch %s", err.Error())
		return false
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
		logp.Err("failed to parse JSON response: %v", err)
		return false
	}

	if !response.Features.ILM.Available || !response.Features.ILM.Enabled {
		logp.Warn(ilmNotSupported)
		return false
	}

	return true
}

const (
	ilmNotSupported          = "current Elasticsearch setup does not qualify for ILM feature"
	noElasticsearchClientSet = "no Elasticsearch client is set"
)
