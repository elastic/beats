// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/server"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

const defaultActionTimeout = 1 * time.Minute

type handlerAppAction struct {
	log *logger.Logger
	srv *server.Server
}

func (h *handlerAppAction) Handle(ctx context.Context, a action, acker fleetAcker) error {
	h.log.Debugf("handlerAppAction: action '%+v' received", a)
	action, ok := a.(*fleetapi.ActionApp)
	if !ok {
		return fmt.Errorf("invalid type, expected ActionApp and received %T", a)
	}

	appState, ok := h.srv.FindByInputType(action.InputType)
	if !ok {
		return fmt.Errorf("matching app is not found for action input: %s", action.InputType)
	}

	params, err := action.MarshalMap()
	if err != nil {
		return err
	}

	start := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := appState.PerformAction(action.InputType, params, defaultActionTimeout)
	end := time.Now().UTC().Format(time.RFC3339Nano)
	if err != nil {
		action.StartedAt = start
		action.CompletedAt = end
		action.Error = err.Error()
	} else {
		action.StartedAt = readMapString(res, "started_at", start)
		action.CompletedAt = readMapString(res, "completed_at", end)
		action.Error = readMapString(res, "error", "")
	}

	return acker.Ack(ctx, action)
}

func readMapString(m map[string]interface{}, key string, def string) string {
	if m == nil {
		return def
	}

	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return def
}
