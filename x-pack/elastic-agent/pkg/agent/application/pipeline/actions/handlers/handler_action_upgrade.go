// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package handlers

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/upgrade"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage/store"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

// Upgrade is a handler for UPGRADE action.
// After running Upgrade agent should download its own version specified by action
// from repository specified by fleet.
type Upgrade struct {
	log      *logger.Logger
	upgrader *upgrade.Upgrader
}

// NewUpgrade creates a new Upgrade handler.
func NewUpgrade(log *logger.Logger, upgrader *upgrade.Upgrader) *Upgrade {
	return &Upgrade{
		log:      log,
		upgrader: upgrader,
	}
}

// Handle handles UPGRADE action.
func (h *Upgrade) Handle(ctx context.Context, a fleetapi.Action, acker store.FleetAcker) error {
	h.log.Debugf("handlerUpgrade: action '%+v' received", a)
	action, ok := a.(*fleetapi.ActionUpgrade)
	if !ok {
		return fmt.Errorf("invalid type, expected ActionUpgrade and received %T", a)
	}

	_, err := h.upgrader.Upgrade(ctx, &upgradeAction{action}, true, false)
	return err
}

type upgradeAction struct {
	*fleetapi.ActionUpgrade
}

func (a *upgradeAction) Version() string {
	return a.ActionUpgrade.Version
}

func (a *upgradeAction) SourceURI() string {
	return a.ActionUpgrade.SourceURI
}

func (a *upgradeAction) FleetAction() *fleetapi.ActionUpgrade {
	return a.ActionUpgrade
}
