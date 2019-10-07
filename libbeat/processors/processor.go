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
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

const logName = "processors"

// Processors is
type Processors struct {
	List []Processor
	log  *logp.Logger
}

type Processor interface {
	Run(event *beat.Event) (*beat.Event, error)
	String() string
}

// NewList creates a new empty processor list.
// Additional processors can be added to the List field.
func NewList(log *logp.Logger) *Processors {
	if log == nil {
		log = logp.NewLogger(logName)
	}
	return &Processors{log: log}
}

// New creates a list of processors from a list of free user configurations.
func New(config PluginConfig) (*Processors, error) {
	procs := NewList(nil)

	for _, procConfig := range config {
		// Handle if/then/else processor which has multiple top-level keys.
		if procConfig.HasField("if") {
			p, err := NewIfElseThenProcessor(procConfig)
			if err != nil {
				return nil, errors.Wrap(err, "failed to make if/then/else processor")
			}
			procs.add(p)
			continue
		}

		if len(procConfig.GetFields()) != 1 {
			return nil, errors.Errorf("each processor must have exactly one "+
				"action, but found %d actions (%v)",
				len(procConfig.GetFields()),
				strings.Join(procConfig.GetFields(), ","))
		}

		actionName := procConfig.GetFields()[0]
		actionCfg, err := procConfig.Child(actionName, -1)
		if err != nil {
			return nil, err
		}

		gen, exists := registry.reg[actionName]
		if !exists {
			var validActions []string
			for k := range registry.reg {
				validActions = append(validActions, k)

			}
			return nil, errors.Errorf("the processor action %s does not exist. Valid actions: %v", actionName, strings.Join(validActions, ", "))
		}

		actionCfg.PrintDebugf("Configure processor action '%v' with:", actionName)
		constructor := gen.Plugin()
		plugin, err := constructor(actionCfg)
		if err != nil {
			return nil, err
		}

		procs.add(plugin)
	}

	if len(procs.List) > 0 {
		procs.log.Debugf("Generated new processors: %v", procs)
	}
	return procs, nil
}

func (procs *Processors) add(p Processor) {
	procs.List = append(procs.List, p)
}

// RunBC (run backwards-compatible) applies the processors, by providing the
// old interface based on common.MapStr.
// The event us temporarily converted to beat.Event. By this 'conversion' the
// '@timestamp' field can not be accessed by processors.
// Note: this method will be removed, when the publisher pipeline BC-API is to
//       be removed.
func (procs *Processors) RunBC(event common.MapStr) common.MapStr {
	ret, err := procs.Run(&beat.Event{Fields: event})
	if err != nil {
		procs.log.Debugw("Error in processor pipeline", "error", err)
	}
	if ret == nil {
		return nil
	}
	return ret.Fields
}

func (procs *Processors) All() []beat.Processor {
	if procs == nil || len(procs.List) == 0 {
		return nil
	}

	ret := make([]beat.Processor, len(procs.List))
	for i, p := range procs.List {
		ret[i] = p
	}
	return ret
}

// Run executes the all processors serially and returns the event and possibly
// an error. If the event has been dropped (canceled) by a processor in the
// list then a nil event is returned.
func (procs *Processors) Run(event *beat.Event) (*beat.Event, error) {
	var err error
	for _, p := range procs.List {
		event, err = p.Run(event)
		if err != nil {
			return event, errors.Wrapf(err, "failed applying processor %v", p)
		}
		if event == nil {
			// Drop.
			return nil, nil
		}
	}
	return event, nil
}

func (procs Processors) String() string {
	var s []string
	for _, p := range procs.List {
		s = append(s, p.String())
	}
	return strings.Join(s, ", ")
}
