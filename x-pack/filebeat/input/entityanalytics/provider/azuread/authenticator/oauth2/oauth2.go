// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package oauth2 provides an OAuth2 authenticator for authenticating with
// Azure Active Directory.
package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread/authenticator"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

const (
	defaultEndpoint = "https://login.microsoftonline.com"
)

var (
	defaultScopes = []string{"https://graph.microsoft.com/.default"}
)

// authResponse matches the format of a token response from the login endpoint.
type authResponse struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	ExtExpiresIn int    `json:"ext_expires_in"`

	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ErrorCodes       []int  `json:"error_codes"`
	CorrelationID    string `json:"correlation_id"`
	TraceID          string `json:"trace_id"`
	ErrorURI         string `json:"error_uri"`
}

// conf contains parameters needed to configure the authenticator.
type conf struct {
	ClientID string   `config:"client_id" validate:"required"`
	TenantID string   `config:"tenant_id" validate:"required"`
	Secret   string   `config:"secret" validate:"required"`
	Endpoint string   `config:"login_endpoint"`
	Scopes   []string `config:"login_scopes"`

	Transport httpcommon.HTTPTransportSettings `config:",inline"`
}

// oauth2 manages retrieving an oauth2 token.
type oauth2 struct {
	conf    conf
	token   string
	expires time.Time
	logger  *logp.Logger
	client  *http.Client
}

// renewToken fetches a new token from the login endpoint.
func (a *oauth2) renewToken(ctx context.Context) error {
	endpointURL, err := url.Parse(a.conf.Endpoint + "/" + a.conf.TenantID + "/oauth2/v2.0/token")
	if err != nil {
		return fmt.Errorf("unable to parse URL: %w", err)
	}
	reqValues := url.Values{
		"client_id":     []string{a.conf.ClientID},
		"scope":         a.conf.Scopes,
		"client_secret": []string{url.QueryEscape(a.conf.Secret)},
		"grant_type":    []string{"client_credentials"},
	}
	reqEncoded := reqValues.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL.String(), strings.NewReader(reqEncoded))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("auth token request failed: %w", err)
	}
	defer res.Body.Close()
	resData, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("unable to read token response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("token request returned unexpected status code: %s, body: %s", res.Status, string(resData))
	}

	var authRes authResponse
	if err = json.Unmarshal(resData, &authRes); err != nil {
		return fmt.Errorf("unable to unmarshal token reseponse: %w", err)
	}

	a.token = authRes.AccessToken
	a.expires = time.Now().Add(time.Duration(authRes.ExpiresIn) * time.Second)
	a.logger.Debugf("Renewed bearer token, expires at: %v", a.expires)

	return nil
}

// Token retrieves the current token.
func (a *oauth2) Token(ctx context.Context) (string, error) {
	if time.Now().Before(a.expires) && a.token != "" {
		a.logger.Debug("Retrieving cached token")
		return a.token, nil
	}

	a.logger.Debugf("Existing token has expired or not set, renewing token")
	if err := a.renewToken(ctx); err != nil {
		return "", fmt.Errorf("failed to renew token: %w", err)
	}

	return a.token, nil
}

// SetLogger sets the logger on this authenticator.
func (a *oauth2) SetLogger(logger *logp.Logger) {
	a.logger = logger
}

// New creates a new OAuth2 authenticator.
func New(cfg *config.C, logger *logp.Logger) (authenticator.Authenticator, error) {
	var c conf
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("unable to unpack oauth2 Authenticator config: %w", err)
	}

	client, err := c.Transport.Client()
	if err != nil {
		return nil, fmt.Errorf("unable to create HTTP client: %w", err)
	}

	a := oauth2{
		conf:   c,
		logger: logger,
		client: client,
	}
	if a.conf.Endpoint == "" {
		a.conf.Endpoint = defaultEndpoint
	}
	if len(a.conf.Scopes) == 0 {
		a.conf.Scopes = defaultScopes
	}

	return &a, nil
}
