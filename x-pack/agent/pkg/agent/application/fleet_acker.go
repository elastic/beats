// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"fmt"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/fleetapi"
	"github.com/elastic/beats/x-pack/agent/pkg/scheduler"
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
		agentInfo: agentInfo,
	}, nil
}

func (f *actionAcker) Ack(action fleetapi.Action) error {
	// checkin
	cmd := fleetapi.NewCheckinCmd(f.agentInfo, f.client)
	req := &fleetapi.CheckinRequest{
		Events: []fleetapi.SerializableEvent{
			fleetapi.Ack(action),
		},
	}

	_, err := cmd.Execute(req)
	if err != nil {
		return errors.New(err, fmt.Sprintf("acknowledge action '%s' failed", action.ID()), errors.TypeNetwork)
	}

	return nil
}

func (f *actionAcker) AckBatch(actions []fleetapi.Action) error {
	// checkin
	events := make([]fleetapi.SerializableEvent, 0, len(actions))
	for _, action := range actions {
		events = append(events, fleetapi.Ack(action))
	}

	cmd := fleetapi.NewCheckinCmd(f.agentInfo, f.client)
	req := &fleetapi.CheckinRequest{
		Events: events,
	}

	_, err := cmd.Execute(req)
	if err != nil {
		return errors.New(err, fmt.Sprintf("acknowledge %d actions '%v' failed", len(actions), actions), errors.TypeNetwork)
	}

	return nil
}

func (f *actionAcker) Commit() error {
	return nil
}

type noopAcker struct{}

func newNoopAcker() *noopAcker {
	return &noopAcker{}
}

func (f *noopAcker) Ack(action fleetapi.Action) error {
	return nil
}

func (*noopAcker) Commit() error { return nil }

var _ fleetAcker = &actionAcker{}
var _ fleetAcker = &noopAcker{}
