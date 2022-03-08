// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package lazy

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

type batchAcker interface {
	AckBatch(ctx context.Context, actions []fleetapi.Action) error
}

type ackForcer interface {
	ForceAck()
}

// Acker is a lazy acker which performs HTTP communication on commit.
type Acker struct {
	log   *logger.Logger
	acker batchAcker
	queue []fleetapi.Action
}

// NewAcker creates a new lazy acker.
func NewAcker(baseAcker batchAcker, log *logger.Logger) *Acker {
	return &Acker{
		acker: baseAcker,
		queue: make([]fleetapi.Action, 0),
		log:   log,
	}
}

// Ack acknowledges action.
func (f *Acker) Ack(ctx context.Context, action fleetapi.Action) error {
	f.enqueue(action)

	if _, isAckForced := action.(ackForcer); isAckForced {
		return f.Commit(ctx)
	}

	return nil
}

// Commit commits ack actions.
func (f *Acker) Commit(ctx context.Context) error {
	err := f.acker.AckBatch(ctx, f.queue)
	if err != nil {
		// do not cleanup on error
		return err
	}

	f.queue = make([]fleetapi.Action, 0)
	return nil
}

func (f *Acker) enqueue(action fleetapi.Action) {
	for _, a := range f.queue {
		if a.ID() == action.ID() {
			f.log.Debugf("action with id '%s' has already been queued", action.ID())
			return
		}
	}
	f.queue = append(f.queue, action)
	f.log.Debugf("appending action with id '%s' to the queue", action.ID())
}
