// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchauth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionauth"
	"go.opentelemetry.io/collector/extension/extensioncapabilities"
)

var (
	_ EndpointsProvider               = (*authenticator)(nil)
	_ extension.Extension             = (*authenticator)(nil)
	_ extensionauth.HTTPClient        = (*authenticator)(nil)
	_ extensioncapabilities.Dependent = (*authenticator)(nil)
)

// EndpointsProvider provides configured Elasticsearch endpoints.
type EndpointsProvider interface {
	Endpoints() []string
}

// authenticator applies Elasticsearch destination credentials and headers over
// the base transport supplied by the consumer to RoundTripper, optionally after
// that transport has been processed by a nested HTTP client authenticator.
type authenticator struct {
	config     *Config
	nestedAuth extensionauth.HTTPClient
}

func newAuthenticator(config *Config) *authenticator {
	return &authenticator{config: config}
}

// Dependencies implements [extensioncapabilities.Dependent].
func (a *authenticator) Dependencies() []component.ID {
	if !a.config.Auth.HasValue() {
		return nil
	}

	return []component.ID{a.config.Auth.Get().AuthenticatorID}
}

// Start resolves the optional nested authenticator without constructing its
// transport.
func (a *authenticator) Start(ctx context.Context, host component.Host) error {
	a.nestedAuth = nil
	if !a.config.Auth.HasValue() {
		return nil
	}

	authConfig := a.config.Auth.Get()
	nestedAuth, err := authConfig.GetHTTPClientAuthenticator(ctx, host.GetExtensions())
	if err != nil {
		return err
	}
	a.nestedAuth = nestedAuth

	return nil
}

// Shutdown is no-op
func (*authenticator) Shutdown(context.Context) error {
	return nil
}

// Endpoints returns a copy of the configured endpoints,
// callers own the returned slice.
func (a *authenticator) Endpoints() []string {
	return slices.Clone(a.config.Endpoints)
}

// RoundTripper optionally delegates transport construction to the nested
// authenticator, then applies Elasticsearch destination headers and credentials.
// If base is nil, http.DefaultTransport is used.
func (a *authenticator) RoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	if base == nil {
		base = http.DefaultTransport
	}
	transport := base

	if a.nestedAuth != nil {
		var err error
		transport, err = a.nestedAuth.RoundTripper(base)
		if err != nil {
			return nil, fmt.Errorf("cannot get roundTripper from nested auth: %w", err)
		}
		if transport == nil {
			return nil, errors.New("nested authenticator returned a nil transport")
		}
	}

	return &authenticatedRoundTripper{
		transport: transport,
		config:    a.config,
	}, nil
}

type authenticatedRoundTripper struct {
	transport http.RoundTripper
	config    *Config
}

func (a *authenticatedRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	clonedRequest := request.Clone(request.Context())

	if clonedRequest.Header == nil {
		clonedRequest.Header = make(http.Header)
	}

	if host, found := a.config.Headers.Get("Host"); found && host != "" {
		clonedRequest.Host = string(host)
	}
	for name, value := range a.config.Headers.Iter {
		clonedRequest.Header.Set(name, string(value))
	}

	if a.config.APIKey != "" {
		encodedAPIKey := base64.StdEncoding.EncodeToString([]byte(a.config.APIKey))
		clonedRequest.Header.Set("Authorization", "ApiKey "+encodedAPIKey)
	} else if a.config.User != "" {
		clonedRequest.SetBasicAuth(a.config.User, string(a.config.Password))
	}

	return a.transport.RoundTrip(clonedRequest)
}

// CloseIdleConnections closes idle connections held by the wrapped transport.
func (a *authenticatedRoundTripper) CloseIdleConnections() {
	if closer, ok := a.transport.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}
