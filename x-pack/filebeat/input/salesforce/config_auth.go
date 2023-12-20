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

	URL            string `config:"url"`
	ClientId       string `config:"client.id"`
	ClientUsername string `config:"client.username"`
	ClientKeyPath  string `config:"client.key_path"`
}

// isEnabled returns true if the `enable` field is set to true in the yaml.
func (o *oAuth2Config) isEnabled() bool {
	return o != nil && (o.Enabled != nil && *o.Enabled)
}

// Validate checks if oauth2 config is valid.
func (o *oAuth2Config) Validate() error {
	if !o.isEnabled() {
		return nil
	}

	switch {
	case o.TokenURL == "":
		return errors.New("token_url must be provided")
	case o.ClientID == "":
		return errors.New("client.id must be provided")
	case o.ClientSecret == "":
		return errors.New("client.secret must be provided")
	case o.Password == "":
		return errors.New("password must be provided")
	case o.User == "":
		return errors.New("user must be provided")
	}

	return nil
}

// isEnabled returns true if the `enable` field is set to true in the yaml.
func (o *jwtConfig) isEnabled() bool {
	return o != nil && (o.Enabled != nil && *o.Enabled)
}

func (o *jwtConfig) Validate() error {
	if !o.isEnabled() {
		return nil
	}

	switch {
	case o.URL == "":
		return errors.New("url must be provided")
	case o.ClientId == "":
		return errors.New("client.id must be provided")
	case o.ClientUsername == "":
		return errors.New("client.username must be provided")
	case o.ClientKeyPath == "":
		return errors.New("client.key_path must be provided")
	}

	return nil
}
