// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package handlers

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage/store"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

// PolicyReassign handles policy reassign change coming from fleet.
type PolicyReassign struct {
	log *logger.Logger
}

// NewPolicyReassign creates a new PolicyReassign handler.
func NewPolicyReassign(log *logger.Logger) *PolicyReassign {
	return &PolicyReassign{
		log: log,
	}
}

// Handle handles POLICY_REASSIGN action.
func (h *PolicyReassign) Handle(ctx context.Context, a fleetapi.Action, acker store.FleetAcker) error {
	h.log.Debugf("handlerPolicyReassign: action '%+v' received", a)

	if err := acker.Ack(ctx, a); err != nil {
		h.log.Errorf("failed to acknowledge POLICY_REASSIGN action with id '%s'", a.ID)
	} else if err := acker.Commit(ctx); err != nil {
		h.log.Errorf("failed to commit acker after acknowledging action with id '%s'", a.ID)
	}

	return nil
}
