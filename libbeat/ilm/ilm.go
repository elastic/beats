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
	"io/ioutil"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

type ILMManager interface {
	Enabled() (bool, error)
	TemplateInfo() Template
}

type Template struct {
	Alias      string
	Pattern    string
	PolicyName string
}

type noopManager struct{}

type singleManager struct {
	cfg        Config
	policyName string
	policy     common.MapStr
}

func NoopManager(info beat.Info, config *common.Config) (ILMManager, error) {
	return (*noopManager)(nil), nil
}

func DefaultManager(info beat.Info, config *common.Config) (ILMManager, error) {
	cfg := struct {
		ILM Config `config:"setup.ilm"` // read ILM settings from setup.ilm namespace
	}{defaultConfig(info)}
	if err := config.Unpack(&cfg); err != nil {
		return nil, err
	}

	if cfg.ILM.Mode == ModeDisabled {
		return NoopManager(info, config)
	}

	var policy common.MapStr
	if path := cfg.ILM.PolicyFile; path != "" {
		contents, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read policy file '%v'", path)
		}

		if err := json.Unmarshal(contents, &policy); err != nil {
			return nil, errors.Wrapf(err, "failed to decode policy file '%v'", path)
		}

	} else {
		policy = ilmDefaultPolicy
	}

	// TODO: create policyName

	return &singleManager{
		cfg:    cfg.ILM,
		policy: policy,
	}, errors.New("TODO")
}

func (*noopManager) Enabled() (bool, error) { return false, nil }
func (*noopManager) TemplateInfo() Template { return Template{} }

func (m *singleManager) Enabled() (bool, error) {
	return false, errors.New("TODO")
}

func (m *singleManager) TemplateInfo() Template {
	alias := m.cfg.RolloverAlias
	return Template{
		Alias:      alias,
		Pattern:    fmt.Sprintf("%s-*", alias),
		PolicyName: m.policyName,
	}
}
