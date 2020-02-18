// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"

	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
)

type handlerUnknown struct {
	log *logger.Logger
}

func (h *handlerUnknown) Handle(_ context.Context, a action, acker fleetAcker) error {
	h.log.Errorf("HandlerUnknown: action '%+v' received", a)
	return nil
}
