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
	"net/url"
	"path"

	"github.com/elastic/beats/libbeat/common"
)

type esClientHandler struct {
	client ESClient
}

var (
	esMinILMVersion       = common.MustNewVersion("6.6.0")
	esMinDefaultILMVesion = common.MustNewVersion("7.0.0")
)

const (
	// esFeaturesPath is used to query Elasticsearch for availability of licensed
	// features.
	esFeaturesPath = "/_xpack"

	esILMPath = "/_ilm/policy"

	esAliasPath = "/_alias"
)

// ESClientHandler creates a new APIHandler executing ILM, and alias queries
// against Elasticsearch.
func ESClientHandler(client ESClient) APIHandler {
	if client == nil {
		return nil
	}
	return &esClientHandler{client}
}

// ESClient defines the minimal interface required for the ESClientHandler to
// prepare a policy and write alias.
type ESClient interface {
	GetVersion() common.Version
	Request(
		method, path string,
		pipeline string,
		params map[string]string,
		body interface{},
	) (int, []byte, error)
}

func (h *esClientHandler) ILMEnabled(mode Mode) (bool, error) {
	if mode == ModeDisabled {
		return false, nil
	}

	avail, probe := h.checkILMVersion(mode)
	if !avail {
		if mode == ModeEnabled {
			ver := h.client.GetVersion()
			return false, errf(ErrESVersionNotSupported,
				"Elasticsearch %v does not support ILM", ver.String())
		}
		return false, nil
	}

	if !probe {
		// version potentially supports ILM, but mode + version indicates that we
		// want to disable ILM support.
		return false, nil
	}

	avail, enabled, err := h.checkILMSupport()
	if err != nil {
		return false, err
	}

	if !avail {
		if mode == ModeEnabled {
			return false, errOf(ErrESVersionNotSupported)
		}
		return false, nil
	}

	if !enabled && mode == ModeEnabled {
		return false, errOf(ErrESILMDisabled)
	}
	return enabled, nil
}

func (h *esClientHandler) CreateILMPolicy(policy Policy) error {
	path := path.Join(esILMPath, policy.Name)
	_, _, err := h.client.Request("PUT", path, "", nil, policy.Body)
	return err
}

func (h *esClientHandler) HasILMPolicy(name string) (bool, error) {
	// XXX: HEAD method does currently not work for checking if a policy exists
	path := path.Join(esILMPath, name)
	status, b, err := h.client.Request("GET", path, "", nil, nil)
	if err != nil && status != 404 {
		return false, wrapErrf(err, ErrRequestFailed,
			"failed to check for policy name '%v': (status=%v) %s", name, status, b)
	}
	return status == 200, nil
}

func (h *esClientHandler) HasAlias(name string) (bool, error) {
	path := path.Join(esAliasPath, name)
	status, b, err := h.client.Request("HEAD", path, "", nil, nil)
	if err != nil && status != 404 {
		return false, wrapErrf(err, ErrRequestFailed,
			"failed to check for alias '%v': (status=%v) %s", name, status, b)
	}
	return status == 200, nil
}

func (h *esClientHandler) CreateAlias(alias Alias) error {
	// Escaping because of date pattern
	// This always assume it's a date pattern by sourrounding it by <...>
	firstIndex := fmt.Sprintf("<%s-%s>", alias.Name, alias.Pattern)
	firstIndex = url.PathEscape(firstIndex)

	body := common.MapStr{
		"aliases": common.MapStr{
			alias.Name: common.MapStr{
				"is_write_index": true,
			},
		},
	}

	// Note: actual aliases are accessible via the index
	status, res, err := h.client.Request("PUT", "/"+firstIndex, "", nil, body)
	if status == 400 {
		return errOf(ErrAliasAlreadyExists)
	} else if err != nil {
		return wrapErrf(err, ErrAliasCreateFailed, "failed to create alias: %s", res)
	}

	return nil
}

func (h *esClientHandler) checkILMVersion(mode Mode) (avail, probe bool) {
	ver := h.client.GetVersion()
	avail = !ver.LessThan(esMinILMVersion)
	if avail {
		probe = (mode == ModeEnabled) ||
			(mode == ModeAuto && !ver.LessThan(esMinDefaultILMVesion))
	}

	return avail, probe
}

func (h *esClientHandler) checkILMSupport() (avail, enbaled bool, err error) {
	var response struct {
		Features struct {
			ILM struct {
				Available bool `json:"available"`
				Enabled   bool `json:"enabled"`
			} `json:"ilm"`
		} `json:"features"`
	}
	status, err := h.queryFeatures(&response)
	if status == 400 {
		// If we get a 400, it's assumed to be the OSS version of Elasticsearch
		return false, false, nil
	}
	if err != nil {
		return false, false, wrapErr(err, ErrILMCheckRequestFailed)
	}

	avail = response.Features.ILM.Available
	enbaled = response.Features.ILM.Enabled
	return avail, enbaled, nil
}

func (h *esClientHandler) queryFeatures(to interface{}) (int, error) {
	status, body, err := h.client.Request("GET", esFeaturesPath, "", nil, nil)
	if status >= 400 || err != nil {
		return status, err
	}

	if to != nil {
		if err := json.Unmarshal(body, to); err != nil {
			return status, wrapErrf(err, ErrInvalidResponse, "failed to parse JSON response")
		}
	}
	return status, nil
}

func (h *esClientHandler) access() ESClient {
	return h.client
}
