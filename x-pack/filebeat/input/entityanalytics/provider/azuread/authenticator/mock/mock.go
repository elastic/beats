// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package mock provides a mock authenticator for testing purposes.
package mock

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread/authenticator"
	"github.com/elastic/elastic-agent-libs/logp"
)

// DefaultTokenValue provides a default token value.
const DefaultTokenValue = "test-token"

// mock implements the authenticator.Authenticator interface.
type mock struct {
	tokenValue string
}

// Token returns the stored token value.
func (a *mock) Token(ctx context.Context) (string, error) {
	return a.tokenValue, nil
}

// SetLogger is not used for this implementation.
func (a *mock) SetLogger(_ *logp.Logger) {}

// New creates a new mock authenticator. A token value may be supplied, otherwise
// the default token value will be used.
func New(tokenValue string) authenticator.Authenticator {
	if tokenValue == "" {
		tokenValue = DefaultTokenValue
	}

	return &mock{tokenValue: tokenValue}
}
