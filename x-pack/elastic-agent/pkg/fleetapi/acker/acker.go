// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package acker

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

// Acker is an acker of actions to fleet.
type Acker interface {
	Ack(ctx context.Context, action fleetapi.Action) error
	Commit(ctx context.Context) error
}
