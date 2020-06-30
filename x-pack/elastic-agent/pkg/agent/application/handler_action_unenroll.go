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

type handlerUnenroll struct {
	log        *logger.Logger
	emitter    emitterFunc
	dispatcher programsDispatcher
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

	// TODO: clean action store

	return acker.Ack(ctx, action)
}
