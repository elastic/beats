// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package handlers

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage/store"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

// AppAction is a handler for application actions.
type AppAction struct {
	log *logger.Logger
}

// NewAppAction creates a new AppAction handler.
func NewAppAction(log *logger.Logger) *AppAction {
	return &AppAction{
		log: log,
	}
}

// Handle handles application action.
func (h *AppAction) Handle(ctx context.Context, a fleetapi.Action, acker store.FleetAcker) error {
	h.log.Debugf("handlerAppAction: action '%+v' received", a)
	action, ok := a.(*fleetapi.ActionApp)
	if !ok {
		return fmt.Errorf("invalid type, expected ActionApp and received %T", a)
	}

	_ = action

	// TODO: handle app action

	return nil
}
