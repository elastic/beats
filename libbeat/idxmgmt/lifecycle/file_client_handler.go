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
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// FileClientHandler implements the Loader interface for writing to a file.
type FileClientHandler struct {
	client        FileClient
	info          beat.Info
	cfg           Config
	defaultPolicy mapstr.M
	name          string
	mode          Mode
	policy        Policy
}

// NewFileClientHandler initializes and returns a new FileClientHandler instance.
func NewFileClientHandler(c FileClient, info beat.Info, cfg RawConfig) (*FileClientHandler, error) {
	// half-unpack to distinguish between a config section that's been explicitly enabled,
	// that way we can set a proper default

	if cfg.DSL.Enabled() && cfg.ILM.Enabled() {
		return nil, fmt.Errorf("only one lifecycle management type can be used, but both ILM and DSL are enabled")
	}

	// default to ILM if no configs are set
	lifecycleCfg := DefaultILMConfig(info).ILM
	var err error
	if cfg.DSL.Enabled() {
		lifecycleCfg = DefaultDSLConfig(info).DSL
		err = cfg.DSL.Unpack(&lifecycleCfg)

		// unpack name value separately
		dsName := DefaultDSLName()
		err := cfg.DSL.Unpack(&dsName)
		if err != nil {
			return nil, fmt.Errorf("error unpacking DSL data stream name: %w", err)
		}
		lifecycleCfg.PolicyName = dsName.DataStreamPattern
	} else if cfg.ILM.Enabled() {
		lifecycleCfg = DefaultILMConfig(info).ILM
		err = cfg.ILM.Unpack(&lifecycleCfg)
	} else {
		logp.L().Infof("No lifecycle config has been explicitly enabled, defauling to ILM")
	}

	if err != nil {
		return nil, fmt.Errorf("error unpacking config: %w", err)
	}

	name, err := ApplyStaticFmtstr(info, lifecycleCfg.PolicyName)
	if err != nil {
		return nil, fmt.Errorf("error creating policy name: %w", err)
	}

	// set defaults
	defaultPolicy := DefaultILMPolicy
	mode := ILM

	if cfg.DSL.Enabled() {
		defaultPolicy = DefaultDSLPolicy
		mode = DSL
	}

	policy, err := createPolicy(lifecycleCfg, info, defaultPolicy)
	if err != nil {
		return nil, fmt.Errorf("error creating policy: %w", err)
	}

	return &FileClientHandler{client: c, info: info, cfg: lifecycleCfg,
		defaultPolicy: defaultPolicy, name: name, policy: policy, mode: mode}, nil

}

// CheckExists returns the state of the check_exists config flag
func (h *FileClientHandler) CheckExists() bool {
	return h.cfg.CheckExists
}

// Overwrite returns the state of the overwrite config flag
func (h *FileClientHandler) Overwrite() bool {
	return h.cfg.Enabled
}

// CheckEnabled indicates whether or not lifecycle management is supported for the configured mode and client version.
func (h *FileClientHandler) CheckEnabled() (bool, error) {
	return checkILMEnabled(h.cfg.Enabled, h.client)
}

// CreatePolicy writes given policy to the configured file.
func (h *FileClientHandler) CreatePolicy(policy Policy) error {
	str := fmt.Sprintf("%s\n", policy.Body.StringToPrint())
	if err := h.client.Write("policy", policy.Name, str); err != nil {
		return fmt.Errorf("error printing policy : %w", err)
	}
	return nil
}

// Policy returns the complete policy
func (h *FileClientHandler) Policy() Policy {
	return h.policy
}

// Mode returns the configured instance mode
func (h *FileClientHandler) Mode() Mode {
	return h.mode
}

// IsElasticsearch returns false
func (h *FileClientHandler) IsElasticsearch() bool {
	return false
}

// CreatePolicyFromConfig creates a lifecycle policy from its config and posts it to elasticsearch
func (h *FileClientHandler) CreatePolicyFromConfig() error {
	// only applicable to testing
	if h.cfg.policyRaw != nil {
		return h.CreatePolicy(*h.cfg.policyRaw)
	}

	err := h.CreatePolicy(h.policy)
	if err != nil {
		return fmt.Errorf("error writing policy: %w", err)
	}
	return nil
}

// PolicyName returns the generated policy name.
func (h *FileClientHandler) PolicyName() string {
	return h.name
}

// HasPolicy always returns false.
func (h *FileClientHandler) HasPolicy() (bool, error) {
	return false, nil
}
