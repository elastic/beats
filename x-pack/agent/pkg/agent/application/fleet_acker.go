// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"fmt"
	"time"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/fleetapi"
	"github.com/elastic/beats/x-pack/agent/pkg/scheduler"
)

const (
	EventTypeAction = "ACTION"
	EventSubtypeACK = "ACKNOWLEDGED"
)

type actionAcker struct {
	log        *logger.Logger
	dispatcher dispatcher
	client     clienter
	scheduler  scheduler.Scheduler
	agentInfo  agentInfo
	reporter   fleetReporter
	done       chan struct{}
}

func newActionAcker(
	log *logger.Logger,
	agentInfo agentInfo,
	client clienter,
) (*actionAcker, error) {
	return &actionAcker{
		log:       log,
		client:    client,
		agentInfo: agentInfo, //TODO(ph): this need to be a struct.
	}, nil
}

func (f *actionAcker) Ack(actionID string) error {
	// get events
	event := newActionEvent(actionID)

	// checkin
	cmd := fleetapi.NewCheckinCmd(f.agentInfo, f.client)
	req := &fleetapi.CheckinRequest{
		Events: []fleetapi.SerializableEvent{event},
	}

	_, err := cmd.Execute(req)
	if err != nil {
		return errors.New(err, fmt.Sprintf("acknowledge action '%s' failed", actionID), errors.TypeNetwork)
	}

	return nil
}

func newActionEvent(actionID string) *actionEvent {
	return &actionEvent{
		Typ:      EventTypeAction,
		Subtype:  EventSubtypeACK,
		ActionID: actionID,
		Msg:      fmt.Sprintf("Acknowledged action %s", actionID),
	}
}

type actionEvent struct {
	Typ      string `json:"type"`
	Subtype  string `json:"subtype"`
	ActionID string `json:"action_id"`
	Msg      string `json:"message"`
}

func (a *actionEvent) Type() string       { return a.Typ }
func (*actionEvent) Timestamp() time.Time { return time.Now() }
func (a *actionEvent) Message() string    { return a.Msg }

type noopAcker struct{}

func newNoopAcker() *noopAcker {
	return &noopAcker{}
}

func (f *noopAcker) Ack(actionID string) error {
	return nil
}
