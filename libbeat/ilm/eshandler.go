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

func (h *esClientHandler) ILMEnabled(mode Mode) (bool, error) {
	if mode == ModeDisabled {
		return false, nil
	}

	avail, probe := h.checkILMVersion(mode)
	if !avail {
		if mode == ModeEnabled {
			return false, h.raiseVersionNotSupported()
		}
		return false, nil
	}

	if !probe {
		// version potentially supports ILM, but mode + version indicates that we
		// want to disable ILM support.
		return false, nil
	}

	avail, enbaled, err := h.requILMSupport()
	if err != nil {
		return false, err
	}

	if !avail {
		if mode == ModeEnabled {
			return false, errOf(ErrESVersionNotSupported)
		}
		return false, nil
	}

	if !enbaled && mode == ModeEnabled {
		return false, errOf(ErrESILMDisabled)
	}
	return enbaled, nil
}

func (h *esClientHandler) CreateILMPolicy(name string, policy common.MapStr) error {
	client := h.access()
	_, _, err := client.Request("PUT", "/_ilm/policy/"+name, "", nil, policy)
	return err
}

func (h *esClientHandler) HasILMPolicy(name string) (bool, error) {
	client := h.access()

	// XXX: HEAD method does currently not work for checking if a policy exists
	status, b, err := client.Request("GET", "/_ilm/policy/"+name, "", nil, nil)
	if err != nil && status != 404 {
		return false, wrapErrf(err, ErrRequestFailed,
			"failed to check for policy name '%v': (status=%v) %s", name, status, b)
	}
	return status == 200, nil
}

func (h *esClientHandler) HasAlias(name string) (bool, error) {
	client := h.access()
	status, b, err := client.Request("HEAD", "/_alias/"+name, "", nil, nil)
	if err != nil && status != 404 {
		return false, wrapErrf(err, ErrRequestFailed,
			"failed to check for alias '%v': (status=%v) %s", name, status, b)
	}
	return status == 200, nil
}

func (h *esClientHandler) CreateAlias(name, firstIndex string) error {
	body := common.MapStr{
		"aliases": common.MapStr{
			name: common.MapStr{
				"is_write_index": true,
			},
		},
	}

	client := h.access()
	status, res, err := client.Request("PUT", "/"+firstIndex, "", nil, body)
	if status == 400 {
		return errOf(ErrAliasAlreadyExists)
	} else if err != nil {
		return wrapErrf(err, ErrAliasCreateFailed, "failed to create alias: %s", res)
	}

	return nil
}

func (h *esClientHandler) raiseVersionNotSupported() error {
	client := h.access()
	ver := client.GetVersion()
	return errf(ErrESVersionNotSupported,
		"Elasticsearch %v does not support ILM", ver.String())
}

func (h *esClientHandler) checkILMVersion(mode Mode) (avail, probe bool) {
	client := h.access()

	ver := client.GetVersion()
	avail = !ver.LessThan(esMinILMVersion)
	if avail {
		probe = (mode == ModeEnabled) ||
			(mode == ModeAuto && !ver.LessThan(esMinDefaultILMVesion))
	}

	return avail, probe
}

func (h *esClientHandler) requILMSupport() (avail, enbaled bool, err error) {
	client := h.access()
	code, body, err := client.Request("GET", "/_xpack", "", nil, nil)

	// If we get a 400, it's assumed to be the OSS version of Elasticsearch
	if code == 400 {
		return false, false, nil
	}
	if err != nil {
		return false, false, wrapErr(err, ErrILMCheckRequestFailed)
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
		return false, false, wrapErrf(err, ErrInvalidResponse, "failed to parse JSON response")
	}

	avail = response.Features.ILM.Available
	enbaled = response.Features.ILM.Enabled
	return avail, enbaled, nil
}

func (h *esClientHandler) access() *elasticsearch.Client {
	return (*elasticsearch.Client)(h)
}
