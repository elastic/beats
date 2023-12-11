// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"errors"
)

type authConfig struct {
	OAuth2 *oAuth2Config `config:"oauth2"`
	JWT    *jwtConfig    `config:"jwt"`
}

type oAuth2Config struct {
	Enabled *bool `config:"enabled"`

	ClientID     string `config:"client.id"`
	ClientSecret string `config:"client.secret"`
	Password     string `config:"password"`
	TokenURL     string `config:"token_url"`
	User         string `config:"user"`
}

type jwtConfig struct {
	Enabled *bool `config:"enabled"`

	URL            string
	ClientId       string
	ClientUsername string
	// ClientKey      struct{}
}

// isEnabled returns true if the `enable` field is set to true in the yaml.
func (o *oAuth2Config) isEnabled() bool {
	return o != nil && (o.Enabled == nil || *o.Enabled)
}

// Validate checks if oauth2 config is valid.
func (o *oAuth2Config) Validate() error {
	if !o.isEnabled() {
		return nil
	}

	if o.TokenURL == "" || o.ClientID == "" || o.ClientSecret == "" {
		return errors.New("both token_url and client credentials must be provided")
	}

	if o.Password == "" || o.User == "" {
		return errors.New("both user and password credentials must be provided")
	}

	return nil
}

// isEnabled returns true if the `enable` field is set to true in the yaml.
func (o *jwtConfig) isEnabled() bool {
	return o != nil && (o.Enabled == nil || *o.Enabled)
}

func (o *jwtConfig) Validate() error {
	if !o.isEnabled() {
		return nil
	}

	return nil
}
