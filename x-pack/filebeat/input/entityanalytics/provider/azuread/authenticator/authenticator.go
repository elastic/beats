// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package authenticator provides an interface for authenticating with
// Azure Active Directory.
package authenticator

import (
	"context"

	"github.com/elastic/elastic-agent-libs/logp"
)

// Authenticator provides an interface for authenticating with
// Azure Active Directory.
type Authenticator interface {
	// Token returns a bearer token or an error if a failure occurred.
	Token(ctx context.Context) (string, error)
	// SetLogger sets the logger on this Authenticator.
	SetLogger(logger *logp.Logger)
}
