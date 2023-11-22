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
	"errors"
	"fmt"
	"net/http"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// ESClientHandler implements the Loader interface for talking to ES.
type ESClientHandler struct {
	client        ESClient
	info          beat.Info
	cfg           Config
	defaultPolicy mapstr.M
	putPath       string
	name          string
	policy        Policy
	mode          Mode
}

// NewESClientHandler initializes and returns an ESClientHandler
func NewESClientHandler(c ESClient, info beat.Info, cfg RawConfig) (*ESClientHandler, error) {
	if !cfg.DSL.Enabled() && cfg.ILM.Enabled() && c.IsServerless() {
		return nil, fmt.Errorf("ILM is enabled/configured but %s is connected to a serverless instance; ILM isn't supported on Serverless Elasticsearch. Configure DSL or set setup.ilm.enabled to false", info.Beat)
	}

	if !cfg.ILM.Enabled() && cfg.DSL.Enabled() && !c.IsServerless() {
		return nil, fmt.Errorf("DSL is enabled/configured but %s is connected to a stateful instance; DSL is only supported on Serverless Elasticsearch. Configure ILM or set setup.dsl.enabled to false", info.Beat)
	}

	if cfg.ILM.Enabled() && cfg.DSL.Enabled() {
		return nil, fmt.Errorf("only one lifecycle management type can be used, but both ILM and DSL are enabled")
	}

	// set default based on ES connection, then unpack user config, if set
	lifecycleCfg := Config{}
	var err error
	if c.IsServerless() {
		lifecycleCfg = DefaultDSLConfig(info).DSL
		if cfg.DSL != nil {
			err = cfg.DSL.Unpack(&lifecycleCfg)
		}

		// unpack name value separately
		dsName := DefaultDSLName()
		if cfg.DSL != nil {
			err := cfg.DSL.Unpack(&dsName)
			if err != nil {
				return nil, fmt.Errorf("error unpacking DSL data stream name: %w", err)
			}
		}
		lifecycleCfg.PolicyName = dsName.DataStreamPattern

	} else {
		lifecycleCfg = DefaultILMConfig(info).ILM
		if cfg.ILM != nil {
			err = cfg.ILM.Unpack(&lifecycleCfg)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("error unpacking lifecycle config: %w", err)
	}

	// create name and policy
	name, err := ApplyStaticFmtstr(info, lifecycleCfg.PolicyName)
	if err != nil {
		return nil, fmt.Errorf("error applying format string to policy name: %w", err)
	}
	if name == "" && lifecycleCfg.Enabled {
		return nil, errors.New("could not generate usable policy name from config. Check setup.*.policy_name fields")
	}
	// deal with conflicts between policy name and template name
	// under serverless, it doesn't make sense to have a policy name that differs from the template name
	// if the user has set both to different values, throw a warning, as overwrite operations will probably fail
	if c.IsServerless() {
		if cfg.TemplateName != "" && cfg.TemplateName != name {
			logp.L().Warnf("setup.dsl.data_stream_pattern is %s, but setup.template.name is %s; under serverless, non-default template and DSL pattern names should be the same. Additional updates & overwrites to this config will not work.", name, cfg.TemplateName)
		}
	}

	// set defaults
	defaultPolicy := DefaultILMPolicy
	mode := ILM
	path := fmt.Sprintf("%s/%s", esILMPath, name)

	if c.IsServerless() {
		defaultPolicy = DefaultDSLPolicy
		mode = DSL
		path = fmt.Sprintf("/_data_stream/%s/_lifecycle", name)
	}

	var policy Policy
	if lifecycleCfg.Enabled { // these are-enabled checks should happen elsewhere, but check again here just in case
		policy, err = createPolicy(lifecycleCfg, info, defaultPolicy)
		if err != nil {
			return nil, fmt.Errorf("error creating a lifecycle policy: %w", err)
		}
	}

	return &ESClientHandler{client: c,
		info: info, cfg: lifecycleCfg,
		defaultPolicy: defaultPolicy, name: name, putPath: path, policy: policy, mode: mode}, nil
}

// CheckExists returns the value of the check_exists config flag
func (h *ESClientHandler) CheckExists() bool {
	return h.cfg.CheckExists
}

// Overwrite returns the value of the overwrite config flag
func (h *ESClientHandler) Overwrite() bool {
	return h.cfg.Overwrite
}

// CheckEnabled indicates whether or not ILM is supported for the configured mode and ES instance.
func (h *ESClientHandler) CheckEnabled() (bool, error) {
	return checkILMEnabled(h.cfg.Enabled, h.client)
}

func (h *ESClientHandler) IsElasticsearch() bool {
	return true
}

// HasPolicy queries Elasticsearch to see if policy with given name exists.
func (h *ESClientHandler) HasPolicy() (bool, error) {
	status, b, err := h.client.Request("GET", h.putPath, "", nil, nil)
	if err != nil && status != http.StatusNotFound {
		return false, fmt.Errorf("%w: failed to check for policy name '%v': (status=%v) (err=%w) %s",
			ErrRequestFailed, h.name, status, err, b)
	}
	return status == http.StatusOK, nil
}

// CreatePolicyFromConfig creates a DSL policy from a raw setup config for the beat
func (h *ESClientHandler) CreatePolicyFromConfig() error {
	// check overwrite before we do this
	// normally other upstream components do this check,
	// but might as well do it here
	if !h.cfg.Overwrite {
		found, err := h.HasPolicy()
		if err != nil {
			return fmt.Errorf("error looking for existing policy: %w", err)
		}
		// maintain old behavior, don't return an error
		if found {
			return nil
		}
	}
	// only applicable to testing
	if h.cfg.policyRaw != nil {
		return h.putPolicyToES(h.putPath, *h.cfg.policyRaw)
	}

	err := h.createAndPutPolicy(h.cfg, h.info)
	if err != nil {
		return fmt.Errorf("error creating policy from config: %w", err)
	}
	return nil
}

// PolicyName returns the policy name
func (h *ESClientHandler) PolicyName() string {
	return h.name
}

// Policy returns the full policy
func (h *ESClientHandler) Policy() Policy {
	return h.policy
}

// Mode returns the connected instance mode
func (h *ESClientHandler) Mode() Mode {
	return h.mode
}

// creates a policy from config, then performs the PUT request to ES
func (h *ESClientHandler) createAndPutPolicy(cfg Config, info beat.Info) error {
	err := h.putPolicyToES(h.putPath, h.policy)
	if err != nil {
		return fmt.Errorf("error submitting policy: %w", err)
	}
	return nil
}

// performs the PUT operation to create a policy
func (h *ESClientHandler) putPolicyToES(path string, policy Policy) error {
	retCode, resp, err := h.client.Request("PUT", path, "", nil, policy.Body)
	if retCode >= http.StatusMultipleChoices {
		return fmt.Errorf("error creating lifecycle policy: got %d from elasticsearch: %s", retCode, resp)
	}
	if err != nil {
		return fmt.Errorf("error in lifecycle PUT request: %w", err)
	}
	return nil
}
