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

package processors

import (
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/conditions"
	"github.com/elastic/beats/libbeat/logp"
)

// WhenProcessor is a tuple of condition plus a Processor.
type WhenProcessor struct {
	condition conditions.Condition
	p         Processor
}

// NewConditional returns a constructor suitable for registering when conditionals as a plugin.
func NewConditional(
	ruleFactory Constructor,
) Constructor {
	return func(cfg *common.Config) (Processor, error) {
		rule, err := ruleFactory(cfg)
		if err != nil {
			return nil, err
		}

		return addCondition(cfg, rule)
	}
}

// NewConditionList takes a slice of Config objects and turns them into real Condition objects.
func NewConditionList(config []conditions.Config) ([]conditions.Condition, error) {
	out := make([]conditions.Condition, len(config))
	for i, condConfig := range config {
		cond, err := conditions.NewCondition(&condConfig)
		if err != nil {
			return nil, err
		}

		out[i] = cond
	}
	return out, nil
}

// NewConditionRule returns a processor that will execute the provided processor if the condition is true.
func NewConditionRule(
	config conditions.Config,
	p Processor,
) (Processor, error) {
	cond, err := conditions.NewCondition(&config)
	if err != nil {
		logp.Err("Failed to initialize lookup condition: %v", err)
		return nil, err
	}

	if cond == nil {
		return p, nil
	}
	return &WhenProcessor{cond, p}, nil
}

// Run executes this WhenProcessor.
func (r *WhenProcessor) Run(event *beat.Event) (*beat.Event, error) {
	if !(r.condition).Check(event) {
		return event, nil
	}
	return r.p.Run(event)
}

func (r *WhenProcessor) String() string {
	return fmt.Sprintf("%v, condition=%v", r.p.String(), r.condition.String())
}

func addCondition(
	cfg *common.Config,
	p Processor,
) (Processor, error) {
	if !cfg.HasField("when") {
		return p, nil
	}
	sub, err := cfg.Child("when", -1)
	if err != nil {
		return nil, err
	}

	condConfig := conditions.Config{}
	if err := sub.Unpack(&condConfig); err != nil {
		return nil, err
	}

	return NewConditionRule(condConfig, p)
}
