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

package elasticsearch

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/elastic-agent-libs/config"
)

const (
	dead_letter_marker_field = "deadlettered"
	drop                     = "drop"
	dead_letter_index        = "dead_letter_index"
)

type DropPolicy struct{}

func (d DropPolicy) action() string {
	return drop
}

func (d DropPolicy) index() string {
	panic("drop policy doesn't have an target index")
}

type DeadLetterIndexPolicy struct {
	Index string
}

func (d DeadLetterIndexPolicy) action() string {
	return dead_letter_index
}

func (d DeadLetterIndexPolicy) index() string {
	return d.Index
}

type nonIndexablePolicy interface {
	action() string
	index() string
}

var (
	policyFactories = map[string]policyFactory{
		drop:              newDropPolicy,
		dead_letter_index: newDeadLetterIndexPolicy,
	}
)

func newDeadLetterIndexPolicy(config *config.C) (nonIndexablePolicy, error) {
	cfgwarn.Beta("The non_indexable_policy dead_letter_index is beta.")
	policy := DeadLetterIndexPolicy{}
	err := config.Unpack(&policy)
	if policy.index() == "" {
		return nil, fmt.Errorf("%s policy requires an `index` to be specified specified", dead_letter_index)
	}
	return policy, err
}

func newDropPolicy(*config.C) (nonIndexablePolicy, error) {
	return defaultDropPolicy(), nil
}

func defaultPolicy() nonIndexablePolicy {
	return defaultDropPolicy()
}

func defaultDropPolicy() nonIndexablePolicy {
	return &DropPolicy{}
}

type policyFactory func(config *config.C) (nonIndexablePolicy, error)

func newNonIndexablePolicy(configNamespace *config.Namespace) (nonIndexablePolicy, error) {
	if configNamespace == nil {
		return defaultPolicy(), nil
	}

	policyType := configNamespace.Name()
	factory, ok := policyFactories[policyType]
	if !ok {
		return nil, fmt.Errorf("no such policy type: %s", policyType)
	}

	return factory(configNamespace.Config())
}
