// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cometd

import "fmt"

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
	if o.TokenURL == "" {
		return fmt.Errorf("token_url must be provided")
	}
	if o.ClientID == "" {
		return fmt.Errorf("client.id must be provided")
	}
	if o.ClientSecret == "" {
		return fmt.Errorf("client.secret must be provided")
	}
	if o.User == "" || o.Password == "" {
		return fmt.Errorf("both user and password must be provided")
	}
	return nil
}
