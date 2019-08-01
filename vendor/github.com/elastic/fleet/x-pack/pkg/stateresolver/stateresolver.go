// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stateresolver

import (
	"github.com/elastic/fleet/pkg/release"
	"github.com/elastic/fleet/x-pack/pkg/bus"
	"github.com/elastic/fleet/x-pack/pkg/bus/events"
	"github.com/elastic/fleet/x-pack/pkg/bus/topic"
	"github.com/elastic/fleet/x-pack/pkg/core/logger"
)

const (
	defaultInputTopic  = topic.Configurations
	defaultOutputTopic = topic.StateChanges
)

// StateResolver is a resolver of a config state change
// it subscribes to Config event and publishes StateChange events based on that/
// Based on StateChange event operator know what to do.
type StateResolver struct {
	l        *logger.Logger
	evb      bus.Bus
	inTopic  topic.Topic
	outTopic topic.Topic
}

// NewStateResolver creates a new state resolver.
// State Resolver automatically subscribes to event and starts processing right away.
func NewStateResolver(log *logger.Logger, b bus.Bus) (*StateResolver, error) {
	return NewCustomStateResolver(log, b, defaultInputTopic, defaultOutputTopic)
}

// NewCustomStateResolver allow to modify default event names.
// Default input is: ConfigEvent
// Default output is: StateChangeEvent
func NewCustomStateResolver(log *logger.Logger, b bus.Bus, inputTopic topic.Topic, outputTopic topic.Topic) (*StateResolver, error) {
	s := &StateResolver{
		l:        log,
		evb:      b,
		inTopic:  inputTopic,
		outTopic: outputTopic,
	}

	err := s.evb.Subscribe(s.inTopic, s.configHandler)
	return s, err
}

func (s *StateResolver) configHandler(t topic.Topic, e bus.Event) {
	s.l.Debugf("StateResolver: received event %v", t)
	if t != s.inTopic {
		s.l.Errorf("StateResolver: received event %v is not the same as subscribed event: %v", t, s.inTopic)
		return
	}

	configChangeEvt, ok := e.(*events.ConfigChangedEvent)
	if !ok {
		s.l.Errorf("received event which is not 'ConfigChangedEvent'")
		return
	}

	// TODO: preform diff between current state of the program and desired state
	stepConfig, err := configChangeEvt.Program.Config.Map()
	if err != nil {
		s.l.Errorf("unable to parse program config for program: %s: %+v", configChangeEvt.Program.Spec.Name, err)
		return
	}

	startStep := events.Step{
		ID:      events.StateChangeRun,
		Process: configChangeEvt.Program.Spec.Cmd,
		Version: release.Version(),
		Meta: map[string]interface{}{
			events.MetaConfigKey: stepConfig,
		},
	}

	steps := []events.Step{startStep}

	sce, err := events.NewStateChange(steps)
	if err != nil {
		s.l.Errorf("failed to create state change event: %v", err)
		return
	}

	s.l.Debugf("StateResolver: pushing event %v", s.outTopic)

	_, err = s.evb.Push(s.outTopic, sce)
	if err != nil {
		s.l.Errorf("failed to push state change event: %v", err)
		return
	}

	s.l.Debugf("StateResolver: pushed event %v", s.outTopic)
}
