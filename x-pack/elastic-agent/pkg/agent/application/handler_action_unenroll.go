// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

// After running Unenroll agent is in idle state, non managed non standalone.
// For it to be operational again it needs to be either enrolled or reconfigured.
type handlerUnenroll struct {
	log         *logger.Logger
	emitter     emitterFunc
	dispatcher  programsDispatcher
	closers     []context.CancelFunc
	actionStore *actionStore
}

func (h *handlerUnenroll) Handle(ctx context.Context, a action, acker fleetAcker) error {
	h.log.Debugf("handlerUnenroll: action '%+v' received", a)
	action, ok := a.(*fleetapi.ActionUnenroll)
	if !ok {
		return fmt.Errorf("invalid type, expected ActionUnenroll and received %T", a)
	}

	// Providing empty map will close all pipelines
	noPrograms := make(map[routingKey][]program.Program)
	h.dispatcher.Dispatch(a.ID(), noPrograms)

	if !action.IsDetected {
		// ACK only events comming from fleet
		if err := acker.Ack(ctx, action); err != nil {
			return err
		}

		// commit all acks before quitting.
		if err := acker.Commit(ctx); err != nil {
			return err
		}
	} else if h.actionStore != nil {
		// backup action for future start to avoid starting fleet gateway loop
		h.actionStore.Add(a)
		h.actionStore.Save()
	}

	// close fleet gateway loop
	for _, c := range h.closers {
		c()
	}

	return nil
}
