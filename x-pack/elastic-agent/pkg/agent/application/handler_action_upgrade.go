// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

// After running Upgrade agent should download its own version specified by action
// from repository specified by fleet.
type handlerUpgrade struct {
	log         *logger.Logger
	emitter     emitterFunc
	dispatcher  programsDispatcher
	closers     []context.CancelFunc
	actionStore *actionStore
}

func (h *handlerUpgrade) Handle(ctx context.Context, a action, acker fleetAcker) error {
	h.log.Debugf("handlerUpgrade: action '%+v' received", a)
	_, ok := a.(*fleetapi.ActionUpgrade)
	if !ok {
		return fmt.Errorf("invalid type, expected ActionUpgrade and received %T", a)
	}

	// TODO: download artifact
	// TODO: unpack correctly, skip root (symlink, config...) unpack data/*
	// TODO: change symlink
	// TODO: mark update happened so we can handle grace period
	// TODO: reexec
	return nil
}
