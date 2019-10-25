// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stateresolver

import (
	"sync"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/configrequest"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	uid "github.com/elastic/beats/x-pack/agent/pkg/id"
)

// StateResolver is a resolver of a config state change
// it subscribes to Config event and publishes StateChange events based on that/
// Based on StateChange event operator know what to do.
type StateResolver struct {
	l        *logger.Logger
	curState state
	mu       sync.Mutex
}

// NewStateResolver allow to modify default event names.
func NewStateResolver(log *logger.Logger) (*StateResolver, error) {
	return &StateResolver{
		l: log,
	}, nil
}

// Resolve resolves passed config into one or multiple steps
func (s *StateResolver) Resolve(cfg configrequest.Request) (uid.ID, []configrequest.Step, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState, steps := converge(s.curState, cfg)
	id, err := uid.Generate()
	if err != nil {
		return id, nil, err
	}

	s.l.Infof("New State ID is %s", newState.ShortID())
	s.l.Infof("Converging state requires execution of %d step(s)", len(steps))

	// Keep the should state for next tick.
	s.curState = newState

	return id, steps, nil
}
