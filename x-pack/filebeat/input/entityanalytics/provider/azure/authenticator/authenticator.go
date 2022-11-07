// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package authenticator

import (
	"context"

	"github.com/elastic/elastic-agent-libs/logp"
)

type Authenticator interface {
	Token(ctx context.Context) (string, error)
	SetLogger(logger *logp.Logger)
}
