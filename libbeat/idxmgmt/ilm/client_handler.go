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

// ClientHandler defines the interface between a remote service and the Manager.
type ClientHandler interface {
	CheckILMEnabled(Mode) (bool, error)

	HasAlias(name string) (bool, error)
	CreateAlias(alias Alias) error

	HasILMPolicy(name string) (bool, error)
	CreateILMPolicy(policy Policy) error
}

// ESClientHandler implements the Loader interface for talking to ES.
type ESClientHandler struct {
	client ESClient
}

// ESClient defines the minimal interface required for the Loader to
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

// FileClientHandler implements the Loader interface for writing to a file.
type FileClientHandler struct {
	client FileClient
}

// FileClient defines the minimal interface required for the Loader to
// prepare a policy and write alias.
type FileClient interface {
	GetVersion() common.Version
	Write(component string, name string, body string) error
}

const (
	// esFeaturesPath is used to query Elasticsearch for availability of licensed
	// features.
	esFeaturesPath = "/_xpack"

	esILMPath = "/_ilm/policy"

	esAliasPath = "/_alias"
)

var (
	esMinILMVersion        = common.MustNewVersion("6.6.0")
	esMinDefaultILMVersion = common.MustNewVersion("7.0.0")
)

// NewESClientHandler initializes and returns an ESClientHandler,
func NewESClientHandler(c ESClient) *ESClientHandler {
	return &ESClientHandler{client: c}
}

// NewFileClientHandler initializes and returns a new FileClientHandler instance.
func NewFileClientHandler(c FileClient) *FileClientHandler {
	return &FileClientHandler{client: c}
}

// CheckILMEnabled indicates whether or not ILM is supported for the configured mode and ES instance.
func (h *ESClientHandler) CheckILMEnabled(mode Mode) (bool, error) {
	if mode == ModeDisabled {
		return false, nil
	}

	avail, probe := checkILMVersion(mode, h.client.GetVersion())
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

// CreateILMPolicy loads the given policy to Elasticsearch.
func (h *ESClientHandler) CreateILMPolicy(policy Policy) error {
	path := path.Join(esILMPath, policy.Name)
	_, _, err := h.client.Request("PUT", path, "", nil, policy.Body)
	return err
}

// HasILMPolicy queries Elasticsearch to see if policy with given name exists.
func (h *ESClientHandler) HasILMPolicy(name string) (bool, error) {
	// XXX: HEAD method does currently not work for checking if a policy exists
	path := path.Join(esILMPath, name)
	status, b, err := h.client.Request("GET", path, "", nil, nil)
	if err != nil && status != 404 {
		return false, wrapErrf(err, ErrRequestFailed,
			"failed to check for policy name '%v': (status=%v) %s", name, status, b)
	}
	return status == 200, nil
}

// HasAlias queries Elasticsearch to see if alias exists. If other resource
// with the same name exists, it returns an error.
func (h *ESClientHandler) HasAlias(name string) (bool, error) {
	status, b, err := h.client.Request("GET", esAliasPath+"/"+name, "", nil, nil)
	if err != nil && status != 404 {
		return false, wrapErrf(err, ErrRequestFailed,
			"failed to check for alias '%v': (status=%v) %s", name, status, b)
	}
	if status == 200 {
		return true, nil
	}

	// Alias doesn't exist, check if there is an index with the same name
	status, b, err = h.client.Request("HEAD", "/"+name, "", nil, nil)
	if err != nil && status != 404 {
		return false, wrapErrf(err, ErrRequestFailed,
			"failed to check for alias '%v': (status=%v) %s", name, status, b)
	}
	if status == 200 {
		return false, errf(ErrInvalidAlias,
			"resource '%v' exists, but it is not an alias", name)
	}
	return false, nil
}

// CreateAlias sends request to Elasticsearch for creating alias.
func (h *ESClientHandler) CreateAlias(alias Alias) error {
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
		// HasAlias fails if there is an index with the same name, that is
		// what we want to check here.
		_, err := h.HasAlias(alias.Name)
		if err != nil {
			return err
		}
		return errOf(ErrAliasAlreadyExists)
	} else if err != nil {
		return wrapErrf(err, ErrAliasCreateFailed, "failed to create alias: %s", res)
	}

	return nil
}

func (h *ESClientHandler) checkILMSupport() (avail, enabled bool, err error) {
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
	enabled = response.Features.ILM.Enabled
	return avail, enabled, nil
}

func (h *ESClientHandler) queryFeatures(to interface{}) (int, error) {
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

// CheckILMEnabled indicates whether or not ILM is supported for the configured mode and client version.
func (h *FileClientHandler) CheckILMEnabled(mode Mode) (bool, error) {
	if mode == ModeDisabled {
		return false, nil
	}
	avail, probe := checkILMVersion(mode, h.client.GetVersion())
	if avail {
		return probe, nil
	}
	if mode != ModeEnabled {
		return false, nil
	}
	version := h.client.GetVersion()
	return false, errf(ErrESVersionNotSupported,
		"Elasticsearch %v does not support ILM", version.String())
}

// CreateILMPolicy writes given policy to the configured file.
func (h *FileClientHandler) CreateILMPolicy(policy Policy) error {
	str := fmt.Sprintf("%s\n", policy.Body.StringToPrint())
	if err := h.client.Write("policy", policy.Name, str); err != nil {
		return fmt.Errorf("error printing policy : %v", err)
	}
	return nil
}

// HasILMPolicy always returns false.
func (h *FileClientHandler) HasILMPolicy(name string) (bool, error) {
	return false, nil
}

// CreateAlias is a noop implementation.
func (h *FileClientHandler) CreateAlias(alias Alias) error {
	return nil
}

// HasAlias always returns false.
func (h *FileClientHandler) HasAlias(name string) (bool, error) {
	return false, nil
}

// avail: indicates whether version supports ILM
// probe: in case version potentially supports ILM, check the combination of mode + version
// to indicate whether or not ILM support should be enabled or disabled
func checkILMVersion(mode Mode, ver common.Version) (avail, probe bool) {
	avail = !ver.LessThan(esMinILMVersion)
	if avail {
		probe = (mode == ModeEnabled) ||
			(mode == ModeAuto && !ver.LessThan(esMinDefaultILMVersion))
	}
	return avail, probe
}
