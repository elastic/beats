// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleet

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

type noopAcker struct{}

func NewAcker() *noopAcker {
	return &noopAcker{}
}

func (f *noopAcker) Ack(ctx context.Context, action fleetapi.Action) error {
	return nil
}

func (*noopAcker) Commit(ctx context.Context) error { return nil }
