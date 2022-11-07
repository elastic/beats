// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mock

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azure/authenticator"
	"github.com/elastic/elastic-agent-libs/logp"
)

const DefaultTokenValue = "test-token"

type mock struct {
	tokenValue string
}

func (a *mock) Token(ctx context.Context) (string, error) {
	return a.tokenValue, nil
}

func (a *mock) SetLogger(_ *logp.Logger) {}

func New(tokenValue string) authenticator.Authenticator {
	if tokenValue == "" {
		tokenValue = DefaultTokenValue
	}

	return &mock{tokenValue: tokenValue}
}
