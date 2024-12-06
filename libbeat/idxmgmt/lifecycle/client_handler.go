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

package lifecycle

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/version"
)

// ClientHandler defines the interface between a remote service and the index Manager.
type ClientHandler interface {
	CheckEnabled() (bool, error)
	HasPolicy() (bool, error)
	CreatePolicyFromConfig() error
	PolicyName() string
	Overwrite() bool
	CheckExists() bool
	Policy() Policy
	Mode() Mode
	IsElasticsearch() bool
}

type VersionCheckerClient interface {
	GetVersion() version.V
}

// ESClient defines the minimal interface required for the Loader to
// prepare a policy.
type ESClient interface {
	VersionCheckerClient
	IsServerless() bool
	Request(
		method, path string,
		pipeline string,
		params map[string]string,
		body interface{},
	) (int, []byte, error)
}

// FileClient defines the minimal interface required for the Loader to
// prepare a policy.
type FileClient interface {
	GetVersion() version.V
	Write(component string, name string, body string) error
}

const (
	esILMPath = "/_ilm/policy"
)

var (
	esMinDefaultILMVersion = version.MustNew("7.0.0")
)

/// ============ generic helpers

func checkILMEnabled(enabled bool, c VersionCheckerClient) (bool, error) {
	if !enabled {
		return false, nil
	}

	ver := c.GetVersion()
	if ver.LessThan(esMinDefaultILMVersion) {
		return false, fmt.Errorf("%w: Elasticsearch %v does not support ILM", ErrESVersionNotSupported, ver.String())
	}
	return true, nil
}

func createPolicy(cfg Config, info beat.Info, defaultPolicy mapstr.M) (Policy, error) {
	name, err := ApplyStaticFmtstr(info, cfg.PolicyName)
	if err != nil {
		return Policy{}, errors.New("failed to read ilm policy name")
	}

	policy := Policy{
		Name: name,
		Body: defaultPolicy,
	}
	if path := cfg.PolicyFile; path != "" {
		contents, err := os.ReadFile(path)
		if err != nil {
			return Policy{}, fmt.Errorf("failed to read policy file '%s': %w", path, err)
		}

		var body map[string]interface{}
		if err := json.Unmarshal(contents, &body); err != nil {
			return Policy{}, fmt.Errorf("failed to decode policy file '%s': %w", path, err)
		}

		policy.Body = body
	}
	return policy, nil
}
