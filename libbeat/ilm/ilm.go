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
	"io/ioutil"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/fmtstr"
)

type Supporter interface {
	Mode() Mode
	Template() TemplateSettings
	Policy() common.MapStr
	Manager(h APIHandler) Manager
}

type Manager interface {
	Enabled() (bool, error)
	EnsureAlias() error
	EnsurePolicy(overwrite bool) error
}

type APIHandler interface {
	ILMEnabled(Mode) (bool, error)

	HasAlias(name string) (bool, error)
	CreateAlias(name, firstIndex string) error

	HasILMPolicy(name string) (bool, error)
	CreateILMPolicy(name string, policy common.MapStr) error
}

type TemplateSettings struct {
	Alias      string
	Pattern    string
	PolicyName string
}

func DefaultSupport(info beat.Info, config *common.Config) (Supporter, error) {
	cfg := struct {
		ILM Config `config:"setup.ilm"` // read ILM settings from setup.ilm namespace
	}{defaultConfig(info)}
	if err := config.Unpack(&cfg); err != nil {
		return nil, err
	}

	if cfg.ILM.Mode == ModeDisabled {
		return NoopSupport(info, config)
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
		policy = DefaultPolicy
	}

	name, err := applyStaticFmtstr(info, &cfg.ILM.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read ilm policy name")
	}

	return &ilmSupport{
		cfg:        cfg.ILM,
		policyName: name,
		policy:     policy,
	}, nil
}

func applyStaticFmtstr(info beat.Info, fmt *fmtstr.EventFormatString) (string, error) {
	return fmt.Run(&beat.Event{
		Fields: common.MapStr{
			// beat object was left in for backward compatibility reason for older configs.
			"beat": common.MapStr{
				"name":    info.Beat,
				"version": info.Version,
			},
			"agent": common.MapStr{
				"name":    info.Beat,
				"version": info.Version,
			},
			// For the Beats that have an observer role
			"observer": common.MapStr{
				"name":    info.Beat,
				"version": info.Version,
			},
		},
		Timestamp: time.Now(),
	})
}
