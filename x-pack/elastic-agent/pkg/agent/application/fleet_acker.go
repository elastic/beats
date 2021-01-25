// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/scheduler"
)

const fleetTimeFormat = "2006-01-02T15:04:05.99999-07:00"

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
		agentInfo: agentInfo,
	}, nil
}

func (f *actionAcker) SetClient(client clienter) {
	f.client = client
}

func (f *actionAcker) Ack(ctx context.Context, action fleetapi.Action) error {
	// checkin
	agentID := f.agentInfo.AgentID()
	cmd := fleetapi.NewAckCmd(f.agentInfo, f.client)
	req := &fleetapi.AckRequest{
		Events: []fleetapi.AckEvent{
			constructEvent(action, agentID),
		},
	}

	_, err := cmd.Execute(ctx, req)
	if err != nil {
		return errors.New(err, fmt.Sprintf("acknowledge action '%s' for elastic-agent '%s' failed", action.ID(), agentID), errors.TypeNetwork)
	}

	f.log.Debugf("action with id '%s' was just acknowledged", action.ID())

	return nil
}

func (f *actionAcker) AckBatch(ctx context.Context, actions []fleetapi.Action) error {
	// checkin
	agentID := f.agentInfo.AgentID()
	events := make([]fleetapi.AckEvent, 0, len(actions))
	ids := make([]string, 0, len(actions))
	for _, action := range actions {
		events = append(events, constructEvent(action, agentID))
		ids = append(ids, action.ID())
	}

	cmd := fleetapi.NewAckCmd(f.agentInfo, f.client)
	req := &fleetapi.AckRequest{
		Events: events,
	}

	f.log.Debugf("%d actions with ids '%s' acknowledging", len(ids), strings.Join(ids, ","))

	_, err := cmd.Execute(ctx, req)
	if err != nil {
		return errors.New(err, fmt.Sprintf("acknowledge %d actions '%v' for elastic-agent '%s' failed", len(actions), actions, agentID), errors.TypeNetwork)
	}
	return nil
}

func (f *actionAcker) Commit(ctx context.Context) error {
	return nil
}

func constructEvent(action fleetapi.Action, agentID string) fleetapi.AckEvent {
	return fleetapi.AckEvent{
		EventType: "ACTION_RESULT",
		SubType:   "ACKNOWLEDGED",
		Timestamp: time.Now().Format(fleetTimeFormat),
		ActionID:  action.ID(),
		AgentID:   agentID,
		Message:   fmt.Sprintf("Action '%s' of type '%s' acknowledged.", action.ID(), action.Type()),
	}
}

type noopAcker struct{}

func newNoopAcker() *noopAcker {
	return &noopAcker{}
}

func (f *noopAcker) Ack(ctx context.Context, action fleetapi.Action) error {
	return nil
}

func (*noopAcker) Commit(ctx context.Context) error { return nil }

var _ fleetAcker = &actionAcker{}
var _ fleetAcker = &noopAcker{}
