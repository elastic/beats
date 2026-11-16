// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchstorage

import (
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configauth"
)

var errMissingAuthenticator = errors.New("an Elasticsearch authenticator must be configured")

type Config struct {
	// Auth identifies the Elasticsearch authenticator that provides the
	// destination endpoints and authenticated HTTP transport.
	Auth configauth.Config `mapstructure:"auth"`
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func (c *Config) Validate() error {
	if c.Auth.AuthenticatorID == (component.ID{}) {
		return errMissingAuthenticator
	}
	return nil
}
