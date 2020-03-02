// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
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

func (f *actionAcker) Ack(ctx context.Context, action fleetapi.Action) error {
	// checkin
	cmd := fleetapi.NewAckCmd(f.agentInfo, f.client)
	req := &fleetapi.AckRequest{
		AgentID: f.agentInfo.AgentID(),
		Actions: []string{
			action.ID(),
		},
	}

	_, err := cmd.Execute(ctx, req)
	if err != nil {
		return errors.New(err, fmt.Sprintf("acknowledge action '%s' failed", action.ID()), errors.TypeNetwork)
	}

	return nil
}

func (f *actionAcker) AckBatch(ctx context.Context, actions []fleetapi.Action) error {
	// checkin
	ids := make([]string, 0, len(actions))
	for _, action := range actions {
		ids = append(ids, action.ID())
	}

	cmd := fleetapi.NewAckCmd(f.agentInfo, f.client)
	req := &fleetapi.AckRequest{
		Actions: ids,
	}

	_, err := cmd.Execute(ctx, req)
	if err != nil {
		return errors.New(err, fmt.Sprintf("acknowledge %d actions '%v' failed", len(actions), actions), errors.TypeNetwork)
	}

	return nil
}

func (f *actionAcker) Commit(ctx context.Context) error {
	return nil
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
