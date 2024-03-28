// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import "errors"

type authConfig struct {
	// See: https://help.salesforce.com/s/articleView?id=sf.remoteaccess_oauth_flows.htm&type=5
	// for more information about OAuth2 flows.
	OAuth2 *OAuth2 `config:"oauth2"`
}

type OAuth2 struct {
	// See: https://help.salesforce.com/s/articleView?id=sf.remoteaccess_oauth_username_password_flow.htm&type=5
	UserPasswordFlow *UserPasswordFlow `config:"user_password_flow"`
	// See: https://help.salesforce.com/s/articleView?id=sf.remoteaccess_oauth_jwt_flow.htm&type=5
	JWTBearerFlow *JWTBearerFlow `config:"jwt_bearer_flow"`
}

type UserPasswordFlow struct {
	Enabled *bool `config:"enabled"`

	ClientID     string `config:"client.id"`
	ClientSecret string `config:"client.secret"`
	Password     string `config:"password"`
	TokenURL     string `config:"token_url"`
	Username     string `config:"username"`
}

type JWTBearerFlow struct {
	Enabled *bool `config:"enabled"`

	URL            string `config:"url"`
	ClientID       string `config:"client.id"`
	ClientUsername string `config:"client.username"`
	ClientKeyPath  string `config:"client.key_path"`
}

// isEnabled returns true if the `enable` field is set to true in the yaml.
func (o *UserPasswordFlow) isEnabled() bool {
	return o != nil && (o.Enabled != nil && *o.Enabled)
}

// Validate checks if User Passworld Flow config is valid.
func (o *UserPasswordFlow) Validate() error {
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
	case o.Username == "":
		return errors.New("username must be provided")
	case o.Password == "":
		return errors.New("password must be provided")

	}

	return nil
}

// isEnabled returns true if the `enable` field is set to true in the yaml.
func (o *JWTBearerFlow) isEnabled() bool {
	return o != nil && (o.Enabled != nil && *o.Enabled)
}

// Validate checks if JWT Bearer Flow config is valid.
func (o *JWTBearerFlow) Validate() error {
	if !o.isEnabled() {
		return nil
	}

	switch {
	case o.URL == "":
		return errors.New("url must be provided")
	case o.ClientID == "":
		return errors.New("client.id must be provided")
	case o.ClientUsername == "":
		return errors.New("client.username must be provided")
	case o.ClientKeyPath == "":
		return errors.New("client.key_path must be provided")
	}

	return nil
}
