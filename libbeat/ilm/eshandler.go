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

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
)

type esClientHandler elasticsearch.Client

var (
	esMinILMVersion       = common.MustNewVersion("6.6.0")
	esMinDefaultILMVesion = common.MustNewVersion("7.0.0")
)

func ESClientHandler(client *elasticsearch.Client) APIHandler {
	if client == nil {
		return nil
	}
	return (*esClientHandler)(client)
}

func (h *esClientHandler) HasILM(required bool) (bool, error) {
	if ok, err := h.checkILMVersion(required); !ok || err != nil {
		return ok, err
	}

	avail, enabled, err := h.requestILMSupport()
	if err != nil {
		return false, err
	}

	return h.checkILMEnabled()
}

func (h *esClientHandler) checkILMVersion(required bool) (bool, error) {
	client := h.access()

	ver := client.GetVersion()
	if ver.LessThan(esMinILMVersion) {
		return false, nil
	}

	// If ES version is < min default we do not enable ILM if user set `enabled: auto`.
	if ver.LessThan(esMinDefaultILMVesion) {
		return !check, nil
	}

	return true, nil
}

func (h *esClientHandler) requestILMSupport() (avail, enbaled bool, err error) {
	client := h.access()
	code, body, err := client.Request("GET", "/_xpack", "", nil, nil)

	// If we get a 400, it's assumed to be the OSS version of Elasticsearch
	if code == 400 {
		return false, false, nil
		// fmt.Errorf("ILM feature is not available in this Elasticsearch version")
	}
	if err != nil {
		return false, false, err
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
		return false, false, fmt.Errorf("failed to parse JSON response: %v", err)
	}

	avail = response.Features.ILM.Available
	enbaled = response.Features.ILM.Enabled
	return avail, enbaled, nil
}

func (h *esClientHandler) checkILMAvailable() (bool, error) {

	if !response.Features.ILM.Available {
		return false, fmt.Errorf("ILM feature is not available in Elasticsearch")
	}

	if !response.Features.ILM.Enabled {
		return false, fmt.Errorf("ILM feature is not enabled in Elasticsearch")
	}

	return true, nil
}

func (h *esClientHandler) access() *elasticsearch.Client {
	return (*elasticsearch.Client)(h)
}
