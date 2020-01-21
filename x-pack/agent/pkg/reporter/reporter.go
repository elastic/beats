// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package reporter

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
)

const (
	// EventTypeState is an record type describing application state change
	EventTypeState = "STATE"
	// EventTypeError is an record type describing application error
	EventTypeError = "ERROR"
	// EventTypeActionResult is an record type describing applications result of an action
	EventTypeActionResult = "ACTION_RESULT"

	// EventSubTypeStarting is an event type indicating application is starting
	EventSubTypeStarting = "STARTING"
	// EventSubTypeInProgress is an event type indicating application is in progress
	EventSubTypeInProgress = "IN_PROGRESS"
	// EventSubTypeConfig is an event indicating application config related event.
	EventSubTypeConfig = "CONFIG"
	// EventSubTypeStopping is an event type indicating application is stopping
	EventSubTypeStopping = "STOPPING"
	// EventSubTypeStopped is an event type indicating application is stopped
	EventSubTypeStopped = "STOPPED"
)

type agentInfo interface {
	AgentID() string
}

// Reporter uses multiple backends which needs to be non-blocking
// to report various events.
type Reporter struct {
	info     agentInfo
	backends []Backend

	l *logger.Logger
}

// NewReporter creates a new reporter with provided set of Backends.
func NewReporter(logger *logger.Logger, info agentInfo, backends ...Backend) *Reporter {
	return &Reporter{
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

// OnStarting reports application starting event.
func (r *Reporter) OnStarting(application string) {
	msg := fmt.Sprintf("Application: %s[%s]: State change: STARTING", application, r.info.AgentID())
	rec := generateRecord(EventTypeState, EventSubTypeStarting, msg)
	r.report(rec)
}

// OnRunning reports application running event.
func (r *Reporter) OnRunning(application string) {
	msg := fmt.Sprintf("Application: %s[%s]: State change: IN_PROGRESS", application, r.info.AgentID())
	rec := generateRecord(EventTypeState, EventSubTypeInProgress, msg)
	r.report(rec)
}

// OnFailing reports application failed event.
func (r *Reporter) OnFailing(application string, err error) {
	msg := fmt.Sprintf("Application: %s[%s]: %v", application, r.info.AgentID(), err)
	rec := generateRecord(EventTypeError, EventSubTypeConfig, msg)
	r.report(rec)
}

// OnStopping reports application stopped event.
func (r *Reporter) OnStopping(application string) {
	msg := fmt.Sprintf("Application: %s[%s]: State change: STOPPING", application, r.info.AgentID())
	rec := generateRecord(EventTypeState, EventSubTypeStopping, msg)
	r.report(rec)
}

// OnStopped reports application stopped event.
func (r *Reporter) OnStopped(application string) {
	msg := fmt.Sprintf("Application: %s[%s]: State change: STOPPED", application, r.info.AgentID())
	rec := generateRecord(EventTypeState, EventSubTypeStopped, msg)
	r.report(rec)
}

// OnFatal reports applications fatal event.
func (r *Reporter) OnFatal(application string, err error) {
	msg := fmt.Sprintf("Application: %s[%s]: %v", application, r.info.AgentID(), err)
	rec := generateRecord(EventTypeError, EventSubTypeConfig, msg)
	r.report(rec)
}

func (r *Reporter) report(e event) {
	var err error

	for _, b := range r.backends {
		if er := b.Report(e); er != nil {
			err = multierror.Append(err, er)
		}
	}

	if err != nil {
		r.l.Error(errors.New(err, "failed reporting event"))
	}
}

func generateRecord(eventype, subType, message string) event {
	return event{
		eventype:  eventype,
		subType:   subType,
		timestamp: time.Now(),
		message:   message,
	}
}
