// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage/store"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

type handlerUnknown struct {
	log *logger.Logger
}

func (h *handlerUnknown) Handle(_ context.Context, a fleetapi.Action, acker store.FleetAcker) error {
	h.log.Errorf("HandlerUnknown: action '%+v' received", a)
	return nil
}
