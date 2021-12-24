// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cometd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

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

type credentials struct {
	AccessToken string `json:"access_token"`
	InstanceURL string `json:"instance_url"`
	IssuedAt    string `json:"issued_at"`
	ID          string `json:"id"`
	TokenType   string `json:"token_type"`
	Signature   string `json:"signature"`
}

// Client wraps the given http.Client and returns a new one that will use the oauth authentication.
func (o *oAuth2Config) client() (credentials, error) {
	route := o.TokenURL
	params := url.Values{"grant_type": {"password"},
		"client_id":     {o.ClientID},
		"client_secret": {o.ClientSecret},
		"username":      {o.User},
		"password":      {o.Password}}
	response, err := http.PostForm(route, params)
	if err != nil {
		return credentials{}, fmt.Errorf("error while sending http request: %v", err)
	}
	decoder := json.NewDecoder(response.Body)
	var creds credentials
	if err := decoder.Decode(&creds); err == io.EOF {
		return credentials{}, fmt.Errorf("reached end of response: %v", err)
	} else if err != nil {
		return credentials{}, fmt.Errorf("error while reading response: %v", err)
	} else if creds.AccessToken == "" {
		return credentials{}, fmt.Errorf("unable to fetch access token")
	}
	return creds, nil
}

// Validate checks if oauth2 config is valid.
func (o *oAuth2Config) Validate() error {
	if o.TokenURL == "" || o.ClientID == "" || o.ClientSecret == "" || o.User == "" || o.Password == "" {
		return errors.New("both token_url and client credentials must be provided")
	}
	return nil
}
