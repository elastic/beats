// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

// handlerPolicyReassign handles policy reassign change coming from fleet.
type handlerPolicyReassign struct {
	log *logger.Logger
}

// Handle handles POLICY_REASSIGN action.
func (h *handlerPolicyReassign) Handle(ctx context.Context, a action, acker fleetAcker) error {
	h.log.Debugf("handlerPolicyReassign: action '%+v' received", a)

	if err := acker.Ack(ctx, a); err != nil {
		h.log.Errorf("failed to acknowledge POLICY_REASSIGN action with id '%s'", a.ID)
	} else if err := acker.Commit(ctx); err != nil {
		h.log.Errorf("failed to commit acker after acknowledging action with id '%s'", a.ID)
	}

	return nil
}
