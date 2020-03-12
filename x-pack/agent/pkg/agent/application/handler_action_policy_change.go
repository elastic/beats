// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/fleetapi"
)

type handlerConfigChange struct {
	log     *logger.Logger
	emitter emitterFunc
}

func (h *handlerConfigChange) Handle(ctx context.Context, a action, acker fleetAcker) error {
	h.log.Debugf("handlerConfigChange: action '%+v' received", a)
	action, ok := a.(*fleetapi.ActionConfigChange)
	if !ok {
		return fmt.Errorf("invalid type, expected ActionConfigChange and received %T", a)
	}

	c, err := config.NewConfigFrom(action.Config)
	if err != nil {
		return errors.New(err, "could not parse the configuration from the policy", errors.TypeConfig)
	}

	h.log.Debugf("handlerConfigChange: emit configuration for action %+v", a)
	if err := h.emitter(c); err != nil {
		return err
	}

	return acker.Ack(ctx, action)
}
