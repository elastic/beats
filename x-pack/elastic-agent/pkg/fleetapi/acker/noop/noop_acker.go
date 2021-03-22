// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleet

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

// Acker is a noop acker.
// Methods of these acker do nothing.
type Acker struct{}

// NewAcker creates a new noop acker.
func NewAcker() *Acker {
	return &Acker{}
}

// Ack acknowledges action.
func (f *Acker) Ack(ctx context.Context, action fleetapi.Action) error {
	return nil
}

// Commit commits ack actions.
func (*Acker) Commit(ctx context.Context) error { return nil }
