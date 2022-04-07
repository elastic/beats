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
	"path"

	"github.com/elastic/beats/v8/libbeat/common"
)

// ClientHandler defines the interface between a remote service and the Manager.
type ClientHandler interface {
	CheckILMEnabled(bool) (bool, error)
	HasILMPolicy(name string) (bool, error)
	CreateILMPolicy(policy Policy) error
}

// ESClientHandler implements the Loader interface for talking to ES.
type ESClientHandler struct {
	client ESClient
}

type VersionCheckerClient interface {
	GetVersion() common.Version
}

// ESClient defines the minimal interface required for the Loader to
// prepare a policy.
type ESClient interface {
	VersionCheckerClient
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
// prepare a policy.
type FileClient interface {
	GetVersion() common.Version
	Write(component string, name string, body string) error
}

const (
	esILMPath = "/_ilm/policy"
)

var (
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
func (h *ESClientHandler) CheckILMEnabled(enabled bool) (bool, error) {
	return checkILMEnabled(enabled, h.client)
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

// CheckILMEnabled indicates whether or not ILM is supported for the configured mode and client version.
func (h *FileClientHandler) CheckILMEnabled(enabled bool) (bool, error) {
	return checkILMEnabled(enabled, h.client)
}

func checkILMEnabled(enabled bool, c VersionCheckerClient) (bool, error) {
	if !enabled {
		return false, nil
	}

	ver := c.GetVersion()
	if ver.LessThan(esMinDefaultILMVersion) {
		return false, errf(ErrESVersionNotSupported, "Elasticsearch %v does not support ILM", ver.String())
	}
	return true, nil
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
