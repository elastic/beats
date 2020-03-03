// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/fleetapi"
)

type batchAcker interface {
	AckBatch(ctx context.Context, actions []fleetapi.Action) error
}

type ackForcer interface {
	ForceAck()
}

type lazyAcker struct {
	acker batchAcker
	queue []fleetapi.Action
}

func newLazyAcker(baseAcker batchAcker) *lazyAcker {
	return &lazyAcker{
		acker: baseAcker,
		queue: make([]fleetapi.Action, 0),
	}
}

func (f *lazyAcker) Ack(ctx context.Context, action fleetapi.Action) error {
	f.queue = append(f.queue, action)

	if _, isAckForced := action.(ackForcer); isAckForced {
		return f.Commit(ctx)
	}

	return nil
}

func (f *lazyAcker) Commit(ctx context.Context) error {
	err := f.acker.AckBatch(ctx, f.queue)
	if err != nil {
		// do not cleanup on error
		return err
	}

	f.queue = make([]fleetapi.Action, 0)
	return nil
}

var _ fleetAcker = &lazyAcker{}
