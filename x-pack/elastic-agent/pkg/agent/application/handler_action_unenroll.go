// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

type handlerUnenroll struct {
	log     *logger.Logger
	emitter emitterFunc
}

func (h *handlerUnenroll) Handle(ctx context.Context, a action, acker fleetAcker) error {
	h.log.Debugf("handlerUnenroll: action '%+v' received", a)
	action, ok := a.(*fleetapi.ActionUnenroll)
	if !ok {
		return fmt.Errorf("invalid type, expected ActionUnenroll and received %T", a)
	}

	// executing empty config stops all the running processes
	emptyConfig := make(map[string]interface{})
	c, err := config.NewConfigFrom(emptyConfig)
	if err != nil {
		return errors.New(err, "could not parse the configuration from the policy", errors.TypeConfig)
	}

	h.log.Debugf("handlerUnenroll: emit configuration for action %+v", a)
	if err := h.emitter(c); err != nil {
		return err
	}

	return acker.Ack(ctx, action)
}
