// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cometd

import (
	"errors"
)

type authConfig struct {
	OAuth2 *oAuth2Config `config:"oauth2"`
}

type oAuth2Config struct {
	// common oauth fields
	ClientID     string `config:"client.id"`
	ClientSecret string `config:"client.secret"`
	User         string `config:"user"`
	Password     string `config:"password"`
	TokenURL     string `config:"token_url"`
}

// Validate checks if oauth2 config is valid.
func (o *oAuth2Config) Validate() error {
	if o.TokenURL == "" || o.ClientID == "" || o.ClientSecret == "" || o.User == "" || o.Password == "" {
		return errors.New("both token_url and client credentials must be provided")
	}
	return nil
}
