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

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
)

const logName = "processors"

// Processors is
type Processors struct {
	List []Processor
	log  *logp.Logger
}

// Processor is the interface that all processors must implement
type Processor interface {
	Run(event *beat.Event) (*beat.Event, error)
	String() string
}

// Closer defines the interface for processors that should be closed after using
// them.
// Close() is not part of the Processor interface because implementing this method
// is also a way to indicate that the processor keeps some resource that needs to
// be released or orderly closed.
type Closer interface {
	Close() error
}

// Close closes a processor if it implements the Closer interface
func Close(p Processor) error {
	if closer, ok := p.(Closer); ok {
		return closer.Close()
	}
	return nil
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
			procs.AddProcessor(p)
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

		procs.AddProcessor(plugin)
	}

	if len(procs.List) > 0 {
		procs.log.Debugf("Generated new processors: %v", procs)
	}
	return procs, nil
}

// AddProcessor adds a single Processor to Processors
func (procs *Processors) AddProcessor(p Processor) {
	procs.List = append(procs.List, p)
}

// AddProcessors adds more Processors to Processors
func (procs *Processors) AddProcessors(p Processors) {
	// Subtlety: it is important here that we append the individual elements of
	// p, rather than p itself, even though
	// p implements the processors.Processor interface. This is
	// because the contents of what we return are later pulled out into a
	// processing.group rather than a processors.Processors, and the two have
	// different error semantics: processors.Processors aborts processing on
	// any error, whereas processing.group only aborts on fatal errors. The
	// latter is the most common behavior, and the one we are preserving here for
	// backwards compatibility.
	// We are unhappy about this and have plans to fix this inconsistency at a
	// higher level, but for now we need to respect the existing semantics.
	procs.List = append(procs.List, p.List...)
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

func (procs *Processors) Close() error {
	var errs multierror.Errors
	for _, p := range procs.List {
		err := Close(p)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs.Err()
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
