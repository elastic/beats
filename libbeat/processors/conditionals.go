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
	"strings"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/conditions"
)

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

// WhenProcessor is a tuple of condition plus a Processor.
type WhenProcessor struct {
	condition conditions.Condition
	p         Processor
}

// NewConditionRule returns a processor that will execute the provided processor if the condition is true.
func NewConditionRule(
	config conditions.Config,
	p Processor,
) (Processor, error) {
	cond, err := conditions.NewCondition(&config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize condition")
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

type ifThenElseConfig struct {
	Cond conditions.Config `config:"if"   validate:"required"`
	Then *common.Config    `config:"then" validate:"required"`
	Else *common.Config    `config:"else"`
}

// IfThenElseProcessor executes one set of processors (then) if the condition is
// true and another set of processors (else) if the condition is false.
type IfThenElseProcessor struct {
	cond conditions.Condition
	then *Processors
	els  *Processors
}

// NewIfElseThenProcessor construct a new IfThenElseProcessor.
func NewIfElseThenProcessor(cfg *common.Config) (*IfThenElseProcessor, error) {
	var config ifThenElseConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	cond, err := conditions.NewCondition(&config.Cond)
	if err != nil {
		return nil, err
	}

	newProcessors := func(c *common.Config) (*Processors, error) {
		if c == nil {
			return nil, nil
		}
		if !c.IsArray() {
			return New([]*common.Config{c})
		}

		var pc PluginConfig
		if err := c.Unpack(&pc); err != nil {
			return nil, err
		}
		return New(pc)
	}

	var ifProcessors, elseProcessors *Processors
	if ifProcessors, err = newProcessors(config.Then); err != nil {
		return nil, err
	}
	if elseProcessors, err = newProcessors(config.Else); err != nil {
		return nil, err
	}

	return &IfThenElseProcessor{cond, ifProcessors, elseProcessors}, nil
}

// Run checks the if condition and executes the processors attached to the
// then statement or the else statement based on the condition.
func (p *IfThenElseProcessor) Run(event *beat.Event) (*beat.Event, error) {
	if p.cond.Check(event) {
		return p.then.Run(event)
	} else if p.els != nil {
		return p.els.Run(event)
	}
	return event, nil
}

func (p *IfThenElseProcessor) String() string {
	var sb strings.Builder
	sb.WriteString("if ")
	sb.WriteString(p.cond.String())
	sb.WriteString(" then ")
	sb.WriteString(p.then.String())
	if p.els != nil {
		sb.WriteString(" else ")
		sb.WriteString(p.els.String())
	}
	return sb.String()
}
