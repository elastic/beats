// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package jwtauth

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// Direct returns a token source whose tokens are the JWTs signed by s. No
// request is made. Each token expires shortly before the JWT does, so that a
// JWT is not sent when it is about to be rejected.
//
// The token sources in this package do not cache. Wrap them in
// oauth2.ReuseTokenSource, which does and is safe for concurrent use.
func Direct(s *Signer) oauth2.TokenSource {
	return directSource{s}
}

type directSource struct{ s *Signer }

func (d directSource) Token() (*oauth2.Token, error) {
	now := time.Now()
	signed, err := d.s.Sign(now)
	if err != nil {
		return nil, err
	}
	life := d.s.Lifetime()
	return &oauth2.Token{
		AccessToken: signed,
		TokenType:   "Bearer",
		Expiry:      now.Add(life - min(time.Minute, life/10)),
	}, nil
}

// GrantConfig is the configuration of the token endpoint request made by
// Grant and ClientAssertion.
type GrantConfig struct {
	// TokenURL is the token endpoint.
	TokenURL string

	// ClientID and ClientSecret, if not empty, are sent in the request body.
	ClientID     string
	ClientSecret string

	// Scopes are sent as the scope parameter. It is used only by
	// ClientAssertion.
	Scopes []string

	// Params are additional request body parameters.
	Params url.Values
}

const (
	grantTypeJWT               = "urn:ietf:params:oauth:grant-type:jwt-bearer"
	assertionTypeJWT           = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"
	grantTypeClientCredentials = "client_credentials"
)

// Grant returns a token source that exchanges a JWT signed by s for an
// access token using the OAuth2 JWT bearer grant (RFC 7523 section 2.1).
// If the server rejects the assertion with invalid_grant and sets a Date
// header, the exchange is retried once with the server's time, to allow for a
// skewed local clock. The HTTP client is taken from ctx as for the oauth2
// package.
func Grant(ctx context.Context, s *Signer, cfg GrantConfig) oauth2.TokenSource {
	return exchangeSource{
		ctx: ctx,
		s:   s,
		cfg: cfg,
		params: func(assertion string) url.Values {
			return url.Values{
				"grant_type": {grantTypeJWT},
				"assertion":  {assertion},
			}
		},
	}
}

// ClientAssertion returns a token source that authenticates the client to the
// token endpoint with a JWT signed by s (RFC 7523 section 2.2) and requests
// a token with the client_credentials grant. The HTTP client is taken from ctx
// as for the oauth2 package.
func ClientAssertion(ctx context.Context, s *Signer, cfg GrantConfig) oauth2.TokenSource {
	return exchangeSource{
		ctx: ctx,
		s:   s,
		cfg: cfg,
		params: func(assertion string) url.Values {
			v := url.Values{
				"grant_type":            {grantTypeClientCredentials},
				"client_assertion_type": {assertionTypeJWT},
				"client_assertion":      {assertion},
			}
			if len(cfg.Scopes) != 0 {
				v.Set("scope", strings.Join(cfg.Scopes, " "))
			}
			return v
		},
	}
}

type exchangeSource struct {
	ctx    context.Context
	s      *Signer
	cfg    GrantConfig
	params func(assertion string) url.Values
}

func (e exchangeSource) Token() (*oauth2.Token, error) {
	tok, err := e.exchange(time.Now())
	if err == nil {
		return tok, nil
	}
	var rerr *oauth2.RetrieveError
	if !errors.As(err, &rerr) || rerr.ErrorCode != "invalid_grant" || rerr.Response == nil {
		return nil, err
	}
	serverTime, perr := http.ParseTime(rerr.Response.Header.Get("Date"))
	if perr != nil || serverTime.IsZero() {
		return nil, err
	}
	return e.exchange(serverTime)
}

func (e exchangeSource) exchange(now time.Time) (*oauth2.Token, error) {
	assertion, err := e.s.Sign(now)
	if err != nil {
		return nil, err
	}
	v := e.params(assertion)
	// Reserved keys win over user-provided parameters.
	extra := maps.Clone(e.cfg.Params)
	maps.DeleteFunc(extra, func(k string, _ []string) bool { _, ok := v[k]; return ok })
	maps.Copy(v, extra)

	cc := clientcredentials.Config{
		ClientID:       e.cfg.ClientID,
		ClientSecret:   e.cfg.ClientSecret,
		TokenURL:       e.cfg.TokenURL,
		AuthStyle:      oauth2.AuthStyleInParams,
		EndpointParams: v,
	}
	return cc.TokenSource(e.ctx).Token()
}
