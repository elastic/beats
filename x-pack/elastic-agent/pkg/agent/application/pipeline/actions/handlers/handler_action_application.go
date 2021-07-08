// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage/store"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/server"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

const defaultActionTimeout = time.Minute

// AppAction is a handler for application actions.
type AppAction struct {
	log *logger.Logger
	srv *server.Server
}

// NewAppAction creates a new AppAction handler.
func NewAppAction(log *logger.Logger, srv *server.Server) *AppAction {
	return &AppAction{
		log: log,
		srv: srv,
	}
}

// Handle handles application action.
func (h *AppAction) Handle(ctx context.Context, a fleetapi.Action, acker store.FleetAcker) error {
	h.log.Debugf("handlerAppAction: action '%+v' received", a)
	action, ok := a.(*fleetapi.ActionApp)
	if !ok {
		return fmt.Errorf("invalid type, expected ActionApp and received %T", a)
	}

	appState, ok := h.srv.FindByInputType(action.InputType)
	if !ok {
		// If the matching action is not found ack the action with the error for action result document
		action.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
		action.CompletedAt = action.StartedAt
		action.Error = fmt.Sprintf("matching app is not found for action input: %s", action.InputType)
		return acker.Ack(ctx, action)
	}

	params, err := action.MarshalMap()
	if err != nil {
		return err
	}

	start := time.Now().UTC()
	res, err := appState.PerformAction(action.InputType, params, defaultActionTimeout)
	end := time.Now().UTC()

	startFormatted := start.Format(time.RFC3339Nano)
	endFormatted := end.Format(time.RFC3339Nano)
	if err != nil {
		action.StartedAt = startFormatted
		action.CompletedAt = endFormatted
		action.Error = err.Error()
	} else {
		action.StartedAt = readMapString(res, "started_at", startFormatted)
		action.CompletedAt = readMapString(res, "completed_at", endFormatted)
		action.Error = readMapString(res, "error", "")
	}

	return acker.Ack(ctx, action)
}

func readMapString(m map[string]interface{}, key string, def string) string {
	if m == nil {
		return def
	}

	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return def
}
