// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchauth

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"golang.org/x/net/http2"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionauth"
)

var (
	_ extension.Extension                 = (*authenticator)(nil)
	_ extensionauth.HTTPClient            = (*authenticator)(nil)
	_ EndpointsProvider                   = (*authenticator)(nil)
	_ interface{ CloseIdleConnections() } = (*authenticatedRoundTripper)(nil)
)

// EndpointsProvider provides configured Elasticsearch endpoints.
type EndpointsProvider interface {
	Endpoints() []string
}

// authenticator owns TLS and proxy settings. RoundTripper constructs a fresh,
// consumer-owned transport from those settings and does not inherit them from
// the supplied base transport.
type authenticator struct {
	config    *Config
	tlsConfig *tls.Config
}

func newAuthenticator(config *Config, tlsConfig *tls.Config) *authenticator {
	return &authenticator{config: config, tlsConfig: tlsConfig}
}

// Start is a no-op.
func (*authenticator) Start(context.Context, component.Host) error {
	return nil
}

// Shutdown is a no-op.
func (*authenticator) Shutdown(context.Context) error {
	return nil
}

// Endpoints returns a copy of the configured endpoints, callers own the
// returned slice.
func (a *authenticator) Endpoints() []string {
	return slices.Clone(a.config.Endpoints)
}

// RoundTripper creates a new configured and authenticated transport. The base
// transport is intentionally ignored: this extension is authoritative for TLS
// and proxy settings.
func (a *authenticator) RoundTripper(_ http.RoundTripper) (http.RoundTripper, error) {
	transport, err := a.newTransport()
	if err != nil {
		return nil, err
	}
	return &authenticatedRoundTripper{transport: transport, config: a.config}, nil
}

func (a *authenticator) newTransport() (*http.Transport, error) {
	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("default HTTP transport has unexpected type %T", http.DefaultTransport)
	}
	cc := &a.config.ClientConfig

	transport := defaultTransport.Clone()
	if a.tlsConfig != nil {
		transport.TLSClientConfig = a.tlsConfig.Clone()
	}
	if cc.ReadBufferSize > 0 {
		transport.ReadBufferSize = cc.ReadBufferSize
	}
	if cc.WriteBufferSize > 0 {
		transport.WriteBufferSize = cc.WriteBufferSize
	}

	// Only Keepalive is supported; the deprecated flat keepalive fields
	// (disable_keep_alives, max_idle_conns, max_idle_conns_per_host,
	// idle_conn_timeout) are not used. Keepalive is populated by default
	// (confighttp.NewDefaultClientConfig) and is None when keepalive.enabled
	// is false, which disables keep-alives.
	if keepalive := cc.Keepalive.Get(); keepalive != nil {
		transport.DisableKeepAlives = false
		transport.MaxIdleConns = keepalive.MaxIdleConns
		transport.MaxIdleConnsPerHost = keepalive.MaxIdleConnsPerHost
		transport.IdleConnTimeout = keepalive.IdleConnTimeout
	} else {
		transport.DisableKeepAlives = true
	}
	transport.MaxConnsPerHost = cc.MaxConnsPerHost
	transport.ForceAttemptHTTP2 = cc.ForceAttemptHTTP2

	if cc.ProxyURL != "" {
		proxyURL, err := url.ParseRequestURI(cc.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy_url: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	if cc.HTTP2ReadIdleTimeout > 0 {
		// ConfigureTransports attaches the returned HTTP/2 transport to transport.
		//nolint:staticcheck // SA1019 matches confighttp.
		http2Transport, err := http2.ConfigureTransports(transport)
		if err != nil {
			return nil, fmt.Errorf("configure HTTP/2 transport: %w", err)
		}
		//nolint:staticcheck // SA1019 matches confighttp.ClientConfig.ToClient.
		http2Transport.ReadIdleTimeout = cc.HTTP2ReadIdleTimeout
		//nolint:staticcheck // SA1019 matches confighttp.ClientConfig.ToClient.
		http2Transport.PingTimeout = cc.HTTP2PingTimeout
	}
	return transport, nil
}

type authenticatedRoundTripper struct {
	transport http.RoundTripper
	config    *Config
}

// RoundTrip clones the request, applies configured headers and credentials to
// the clone, and forwards it to the underlying transport.
func (a *authenticatedRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	clonedRequest := request.Clone(request.Context())
	if clonedRequest.Header == nil {
		clonedRequest.Header = make(http.Header)
	}
	for name, value := range a.config.ClientConfig.Headers.Iter {
		// net/http ignores a Host header entry, the request Host field must be set.
		if strings.EqualFold(name, "Host") && value != "" {
			clonedRequest.Host = string(value)
		}
		clonedRequest.Header.Set(name, string(value))
	}
	if a.config.APIKey != "" {
		clonedRequest.Header.Set("Authorization", "ApiKey "+string(a.config.APIKey))
	} else if a.config.User != "" {
		clonedRequest.SetBasicAuth(a.config.User, string(a.config.Password))
	}
	return a.transport.RoundTrip(clonedRequest)
}

// CloseIdleConnections forwards connection-pool cleanup to the consumer-owned
// transport when it supports the standard HTTP close operation.
func (a *authenticatedRoundTripper) CloseIdleConnections() {
	if closer, ok := a.transport.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}
