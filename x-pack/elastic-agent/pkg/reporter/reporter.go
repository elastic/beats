// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package reporter

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
)

const (
	// EventTypeState is an record type describing application state change
	EventTypeState = "STATE"
	// EventTypeError is an record type describing application error
	EventTypeError = "ERROR"
	// EventTypeActionResult is an record type describing applications result of an action
	EventTypeActionResult = "ACTION_RESULT"

	// EventSubTypeStopped is an event type indicating application is stopped.
	EventSubTypeStopped = "STOPPED"
	// EventSubTypeStarting is an event type indicating application is starting.
	EventSubTypeStarting = "STARTING"
	// EventSubTypeInProgress is an event type indicating application is in progress.
	EventSubTypeInProgress = "IN_PROGRESS"
	// EventSubTypeConfig is an event indicating application config related event.
	EventSubTypeConfig = "CONFIG"
	// EventSubTypeRunning is an event indicating application running related event.
	EventSubTypeRunning = "RUNNING"
	// EventSubTypeFailed is an event type indicating application is failed.
	EventSubTypeFailed = "FAILED"
	// EventSubTypeStopping is an event type indicating application is stopping.
	EventSubTypeStopping = "STOPPING"
)

type agentInfo interface {
	AgentID() string
}

// Reporter uses multiple backends which needs to be non-blocking
// to report various events.
type Reporter struct {
	ctx      context.Context
	info     agentInfo
	backends []Backend

	l *logger.Logger
}

// NewReporter creates a new reporter with provided set of Backends.
func NewReporter(ctx context.Context, logger *logger.Logger, info agentInfo, backends ...Backend) *Reporter {
	return &Reporter{
		ctx:      ctx,
		info:     info,
		backends: backends,
		l:        logger,
	}
}

// Close stops the reporter. For further reporting new reporter needs to be created.
func (r *Reporter) Close() {
	for _, c := range r.backends {
		c.Close()
	}
}

// OnStateChange called when state of an application changes.
func (r *Reporter) OnStateChange(id string, name string, state state.State) {
	rec := generateRecord(r.info.AgentID(), id, name, state)
	r.report(r.ctx, rec)
}

func (r *Reporter) report(ctx context.Context, e event) {
	var err error

	for _, b := range r.backends {
		if er := b.Report(ctx, e); er != nil {
			err = multierror.Append(err, er)
		}
	}

	if err != nil {
		r.l.Error(errors.New(err, "failed reporting event"))
	}
}

func generateRecord(agentID string, id string, name string, s state.State) event {
	eventType := EventTypeState

	var subType string
	var subTypeText string
	switch s.Status {
	case state.Stopped:
		subType = EventSubTypeStopped
		subTypeText = EventSubTypeStopped
	case state.Starting:
		subType = EventSubTypeStarting
		subTypeText = EventSubTypeStarting
	case state.Configuring:
		subType = EventSubTypeConfig
		subTypeText = EventSubTypeConfig
	case state.Running:
		subType = EventSubTypeRunning
		subTypeText = EventSubTypeRunning
	case state.Degraded:
		// Fleet doesn't understand degraded
		subType = EventSubTypeRunning
		subTypeText = "DEGRADED"
	case state.Failed:
		eventType = EventTypeError
		subType = EventSubTypeFailed
		subTypeText = EventSubTypeFailed
	case state.Crashed:
		eventType = EventTypeError
		subType = EventSubTypeFailed
		subTypeText = "CRASHED"
	case state.Stopping:
		subType = EventSubTypeStopping
		subTypeText = EventSubTypeStopping
	case state.Restarting:
		subType = EventSubTypeStarting
		subTypeText = "RESTARTING"
	}

	err := errors.New(
		fmt.Errorf(s.Message),
		fmt.Sprintf("Application: %s[%s]: State changed to %s", id, agentID, subTypeText),
		errors.TypeApplication,
		errors.M(errors.MetaKeyAppID, id),
		errors.M(errors.MetaKeyAppName, name))
	var payload map[string]interface{}
	if s.Payload != nil {
		payload = map[string]interface{}{
			name: s.Payload,
		}
	}
	return event{
		eventype:  eventType,
		subType:   subType,
		timestamp: time.Now(),
		message:   err.Error(),
		payload:   payload,
	}
}
