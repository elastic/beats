// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/upgrade"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

// After running Upgrade agent should download its own version specified by action
// from repository specified by fleet.
type handlerUpgrade struct {
	log      *logger.Logger
	upgrader *upgrade.Upgrader
}

func (h *handlerUpgrade) Handle(ctx context.Context, a action, acker fleetAcker) error {
	h.log.Debugf("handlerUpgrade: action '%+v' received", a)
	action, ok := a.(*fleetapi.ActionUpgrade)
	if !ok {
		return fmt.Errorf("invalid type, expected ActionUpgrade and received %T", a)
	}

	return h.upgrader.Upgrade(ctx, &upgradeAction{action}, true)
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
