// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type authConfig struct {
	OAuth2 *oAuth2Config `config:"oauth2"`
}

type oAuth2Config struct {
	Enabled *bool `config:"enabled"`

	// common oauth fields
	ClientID     string `config:"client.id"`
	ClientSecret string `config:"client.secret"`
	Password     string `config:"password"`
	TokenURL     string `config:"token_url"`
	User         string `config:"user"`
}

// isEnabled returns true if the `enable` field is set to true in the yaml.
func (o *oAuth2Config) isEnabled() bool {
	return o != nil && (o.Enabled == nil || *o.Enabled)
}

// clientCredentialsGrant creates http client from token_url and client credentials
// held by the receiver.
func (o *oAuth2Config) clientCredentialsGrant(ctx context.Context, _ *http.Client) *http.Client {
	creds := clientcredentials.Config{
		ClientID:     o.ClientID,
		ClientSecret: maybeString(&o.ClientSecret),
	}
	return creds.Client(ctx)
}

// client wraps the given http.Client and returns a new one that will use the oauth authentication.
func (o *oAuth2Config) client(ctx context.Context, client *http.Client) (*http.Client, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, client)

	if o.User != "" || o.Password != "" {
		conf := &oauth2.Config{
			ClientID:     o.ClientID,
			ClientSecret: maybeString(&o.ClientSecret),
			Endpoint: oauth2.Endpoint{
				TokenURL:  o.TokenURL,
				AuthStyle: oauth2.AuthStyleAutoDetect,
			},
		}

		token, err := conf.PasswordCredentialsToken(ctx, o.User, o.Password)
		if err != nil {
			return nil, fmt.Errorf("oauth2 client: error loading credentials using user and password: %w", err)
		}
		return conf.Client(ctx, token), nil
	}
	return o.clientCredentialsGrant(ctx, client), nil
}

// maybeString returns the string pointed to by p or "" if p in nil.
func maybeString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
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
