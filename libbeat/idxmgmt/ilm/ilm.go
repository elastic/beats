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
	"github.com/elastic/beats/libbeat/logp"
)

// SupportFactory is used to define a policy type to be used.
type SupportFactory func(*logp.Logger, beat.Info, *common.Config) (Supporter, error)

// Supporter implements ILM support. For loading the policies and creating
// write alias a manager instance must be generated.
type Supporter interface {
	Mode() Mode
	Alias() Alias
	Policy() Policy
	Manager(h ClientHandler) Manager
}

// Manager uses a ClientHandler to install a policy.
type Manager interface {
	Enabled() (bool, error)

	EnsureAlias() error

	// EnsurePolicy installs a policy if it does not exist. The policy is always
	// written if overwrite is set.
	// The created flag is set to true only if a new policy is created. `created`
	// is false if an existing policy gets overwritten.
	EnsurePolicy(overwrite bool) (created bool, err error)
}

// Policy describes a policy to be loaded into Elasticsearch.
// See: [Policy phases and actions documentation](https://www.elastic.co/guide/en/elasticsearch/reference/master/ilm-policy-definition.html).
type Policy struct {
	Name string
	Body map[string]interface{}
}

// Alias describes the alias to be created in Elasticsearch.
type Alias struct {
	Name    string
	Pattern string
}

// DefaultSupport configures a new default ILM support implementation.
func DefaultSupport(log *logp.Logger, info beat.Info, config *common.Config) (Supporter, error) {
	if log == nil {
		log = logp.NewLogger("ilm")
	} else {
		log = log.Named("ilm")
	}

	cfg := defaultConfig(info)
	if config != nil {
		if err := config.Unpack(&cfg); err != nil {
			return nil, err
		}
	}

	if cfg.Mode == ModeDisabled {
		return NoopSupport(info, config)
	}

	name, err := applyStaticFmtstr(info, &cfg.PolicyName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read ilm policy name")
	}

	alias := Alias{
		Name:    cfg.RolloverAlias,
		Pattern: cfg.Pattern,
	}

	policy := Policy{
		Name: name,
		Body: DefaultPolicy,
	}
	if path := cfg.PolicyFile; path != "" {
		contents, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read policy file '%v'", path)
		}

		var body map[string]interface{}
		if err := json.Unmarshal(contents, &body); err != nil {
			return nil, errors.Wrapf(err, "failed to decode policy file '%v'", path)
		}

		policy.Body = body
	}

	log.Infof("Policy name: %v", name)
	return NewDefaultSupport(log, cfg.Mode, alias, policy, cfg.Overwrite, cfg.CheckExists), nil
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
