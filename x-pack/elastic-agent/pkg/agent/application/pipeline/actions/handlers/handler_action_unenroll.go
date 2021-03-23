// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package handlers

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/pipeline"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage/store"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

type stateStore interface {
	Add(fleetapi.Action)
	AckToken() string
	SetAckToken(ackToken string)
	Save() error
	Actions() []fleetapi.Action
}

// Unenroll results in  running agent entering idle state, non managed non standalone.
// For it to be operational again it needs to be either enrolled or reconfigured.
type Unenroll struct {
	log        *logger.Logger
	emitter    pipeline.EmitterFunc
	dispatcher pipeline.Router
	closers    []context.CancelFunc
	stateStore stateStore
}

// NewUnenroll creates a new Unenroll handler.
func NewUnenroll(
	log *logger.Logger,
	emitter pipeline.EmitterFunc,
	dispatcher pipeline.Router,
	closers []context.CancelFunc,
	stateStore stateStore,
) *Unenroll {
	return &Unenroll{
		log:        log,
		emitter:    emitter,
		dispatcher: dispatcher,
		closers:    closers,
		stateStore: stateStore,
	}
}

// Handle handles UNENROLL action.
func (h *Unenroll) Handle(ctx context.Context, a fleetapi.Action, acker store.FleetAcker) error {
	h.log.Debugf("handlerUnenroll: action '%+v' received", a)
	action, ok := a.(*fleetapi.ActionUnenroll)
	if !ok {
		return fmt.Errorf("invalid type, expected ActionUnenroll and received %T", a)
	}

	// Providing empty map will close all pipelines
	noPrograms := make(map[pipeline.RoutingKey][]program.Program)
	h.dispatcher.Route(a.ID(), noPrograms)

	if !action.IsDetected {
		// ACK only events comming from fleet
		if err := acker.Ack(ctx, action); err != nil {
			return err
		}

		// commit all acks before quitting.
		if err := acker.Commit(ctx); err != nil {
			return err
		}
	} else if h.stateStore != nil {
		// backup action for future start to avoid starting fleet gateway loop
		h.stateStore.Add(a)
		h.stateStore.Save()
	}

	// close fleet gateway loop
	for _, c := range h.closers {
		c()
	}

	return nil
}
