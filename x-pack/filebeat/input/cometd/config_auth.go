// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cometd

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
)

const authStyleInParams = 1

type authConfig struct {
	OAuth2 *oAuth2Config `config:"oauth2"`
}

type oAuth2Config struct {
	// common oauth fields
	ClientID       string              `config:"client.id"`
	ClientSecret   string              `config:"client.secret"`
	User           string              `config:"user"`
	Password       string              `config:"password"`
	EndpointParams map[string][]string `config:"endpoint_params"`
	Scopes         []string            `config:"scopes"`
	TokenURL       string              `config:"token_url"`
}

// Client wraps the given http.Client and returns a new one that will use the oauth authentication.
func (o *oAuth2Config) client(ctx context.Context, client *http.Client) (*http.Client, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, client)
	conf := &oauth2.Config{
		ClientID:     o.ClientID,
		ClientSecret: o.ClientSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL:  o.TokenURL,
			AuthStyle: authStyleInParams,
		},
	}
	token, err := conf.PasswordCredentialsToken(ctx, o.User, o.Password)
	if err != nil {
		return nil, fmt.Errorf("oauth2 client: error loading credentials using user and password: %w", err)
	}
	return conf.Client(ctx, token), nil
}

// Validate checks if oauth2 config is valid.
func (o *oAuth2Config) Validate() error {
	if o.TokenURL == "" || o.ClientID == "" || o.ClientSecret == "" || o.User == "" || o.Password == "" {
		return errors.New("both token_url and client credentials must be provided")
	}
	return nil
}
