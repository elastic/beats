// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fetcher

import (
	"context"

	"github.com/elastic/elastic-agent-libs/logp"
)

type Fetcher interface {
	Groups(ctx context.Context, deltaLink string) ([]*Group, string, error)
	Users(ctx context.Context, deltaLink string) ([]*User, string, error)
	SetLogger(logger *logp.Logger)
}
