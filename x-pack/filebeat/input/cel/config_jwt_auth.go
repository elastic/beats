// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cel

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"time"

	"golang.org/x/oauth2"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/internal/jwtauth"
)

// JWT authentication modes.
const (
	// jwtModeDirect uses the signed JWT itself as the bearer token.
	jwtModeDirect = "direct"
	// jwtModeGrant exchanges the signed JWT for an access token with the
	// OAuth2 JWT bearer grant.
	jwtModeGrant = "grant"
	// jwtModeClientAssertion authenticates to the token endpoint with the
	// signed JWT and requests a token with the client credentials grant.
	jwtModeClientAssertion = "client_assertion"
)

const (
	// defaultJWTDirectLifetime is below the one hour maximum accepted by
	// Snowflake.
	defaultJWTDirectLifetime = 59 * time.Minute
	// defaultJWTExchangeLifetime is below the 60 second maximum accepted by
	// Box.
	defaultJWTExchangeLifetime = 45 * time.Second
)

// jwtAuthConfig is the configuration for the auth.jwt block.
type jwtAuthConfig struct {
	Enabled *bool `config:"enabled"`

	Mode       string         `config:"mode"`
	PrivateKey string         `config:"private_key"`
	Passphrase string         `config:"passphrase"`
	Algorithm  string         `config:"algorithm"`
	KeyID      string         `config:"key_id"`
	Lifetime   time.Duration  `config:"lifetime"`
	JTI        bool           `config:"jti"`
	Claims     map[string]any `config:"claims"`

	// Token endpoint options for the exchange modes.
	TokenURL       string              `config:"token_url"`
	ClientID       string              `config:"client.id"`
	ClientSecret   string              `config:"client.secret"`
	Scopes         []string            `config:"scopes"`
	EndpointParams map[string][]string `config:"endpoint_params"`
}

// isEnabled returns true if the `enabled` field is set to true in the yaml.
func (j *jwtAuthConfig) isEnabled() bool {
	return j != nil && (j.Enabled == nil || *j.Enabled)
}

// Validate checks that the JWT authentication configuration is coherent.
// The private key is parsed so that a bad key or passphrase is found at
// startup.
func (j *jwtAuthConfig) Validate() error {
	if !j.isEnabled() {
		return nil
	}

	var reserved []string
	switch j.Mode {
	case jwtModeDirect:
		switch {
		case j.TokenURL != "":
			return errors.New("jwt validation error: token_url must not be set when mode is direct")
		case j.ClientID != "" || j.ClientSecret != "":
			return errors.New("jwt validation error: client.id and client.secret must not be set when mode is direct")
		case len(j.Scopes) != 0:
			return errors.New("jwt validation error: scopes must not be set when mode is direct")
		case len(j.EndpointParams) != 0:
			return errors.New("jwt validation error: endpoint_params must not be set when mode is direct")
		}
	case jwtModeGrant:
		reserved = []string{"grant_type", "assertion", "client_id", "client_secret"}
		if len(j.Scopes) != 0 {
			return errors.New("jwt validation error: scopes must not be set when mode is grant; use a scope claim")
		}
	case jwtModeClientAssertion:
		reserved = []string{"grant_type", "client_assertion", "client_assertion_type", "scope", "client_id", "client_secret"}
	case "":
		return errors.New("jwt validation error: mode must be provided")
	default:
		return fmt.Errorf("jwt validation error: unknown mode %q", j.Mode)
	}
	if j.Mode != jwtModeDirect {
		if j.TokenURL == "" {
			return fmt.Errorf("jwt validation error: token_url must be provided when mode is %s", j.Mode)
		}
		for _, k := range reserved {
			if _, ok := j.EndpointParams[k]; ok {
				return fmt.Errorf("jwt validation error: endpoint_params must not include reserved key %q", k)
			}
		}
	}

	if j.PrivateKey == "" {
		return errors.New("jwt validation error: private_key must be provided")
	}
	if iss, _ := j.Claims["iss"].(string); iss == "" {
		return errors.New("jwt validation error: claims.iss must be provided as a non-empty string")
	}
	if j.Lifetime < 0 {
		return errors.New("jwt validation error: lifetime must not be negative")
	}
	_, err := j.signer()
	if err != nil {
		return fmt.Errorf("jwt validation error: %w", err)
	}
	return nil
}

// signer returns the signer described by the configuration.
func (j *jwtAuthConfig) signer() (*jwtauth.Signer, error) {
	key, err := jwtauth.ParsePrivateKey([]byte(j.PrivateKey), j.Passphrase)
	if err != nil {
		return nil, fmt.Errorf("private_key: %w", err)
	}

	claims := maps.Clone(j.Claims)
	if _, ok := claims["aud"]; !ok && j.Mode != jwtModeDirect {
		// The audience of an assertion is conventionally the token endpoint.
		claims["aud"] = j.TokenURL
	}
	life := j.Lifetime
	if life == 0 {
		life = defaultJWTExchangeLifetime
		if j.Mode == jwtModeDirect {
			life = defaultJWTDirectLifetime
		}
	}
	s, err := jwtauth.NewSigner(jwtauth.SignerConfig{
		Key:       key,
		Algorithm: j.Algorithm,
		KeyID:     j.KeyID,
		Claims:    claims,
		Lifetime:  life,
		JTI:       j.JTI,
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}

// client returns a copy of client that authenticates requests with JWTs, and
// refreshes them when they expire.
func (j *jwtAuthConfig) client(ctx context.Context, client *http.Client) (*http.Client, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, client)

	s, err := j.signer()
	if err != nil {
		return nil, fmt.Errorf("jwt client: %w", err)
	}
	grant := jwtauth.GrantConfig{
		TokenURL:     j.TokenURL,
		ClientID:     j.ClientID,
		ClientSecret: j.ClientSecret,
		Scopes:       slices.Clone(j.Scopes),
		Params:       url.Values(maps.Clone(j.EndpointParams)),
	}
	var ts oauth2.TokenSource
	switch j.Mode {
	case jwtModeDirect:
		ts = jwtauth.Direct(s)
	case jwtModeGrant:
		ts = jwtauth.Grant(ctx, s, grant)
	case jwtModeClientAssertion:
		ts = jwtauth.ClientAssertion(ctx, s, grant)
	default:
		return nil, fmt.Errorf("jwt client: unknown mode %q", j.Mode)
	}

	// Get the first token now so that bad credentials fail at startup.
	tok, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("jwt client: initial token: %w", err)
	}
	return oauth2.NewClient(ctx, oauth2.ReuseTokenSource(tok, ts)), nil
}
