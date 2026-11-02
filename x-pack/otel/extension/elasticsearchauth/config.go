// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package elasticsearchauth provides Elasticsearch HTTP authentication for OTel components.
package elasticsearchauth

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configopaque"
)

// Config configures the Elasticsearch authentication extension.
//
// The embedded HTTP configuration supplies the TLS, proxy, header, pool, and
// keepalive settings used by transports returned from RoundTripper. Its
// endpoint, timeout, auth, middleware, cookie, and compression settings are
// intentionally unsupported by this extension.
type Config struct {
	ClientConfig confighttp.ClientConfig `mapstructure:",squash"`

	// Endpoints contains the resolved Elasticsearch HTTP(S) endpoints.
	Endpoints []string `mapstructure:"endpoints"`

	// User configures HTTP Basic authentication with Password.
	User string `mapstructure:"user"`
	// Password configures HTTP Basic authentication with User.
	Password configopaque.String `mapstructure:"password"`
	// APIKey configures Elasticsearch ApiKey authentication. It is sent as is
	// in the Authorization header; its format is not validated.
	APIKey configopaque.String `mapstructure:"api_key"`
}

func createDefaultConfig() component.Config {
	return &Config{ClientConfig: confighttp.NewDefaultClientConfig()}
}

// Validate validates configuration relationships without accessing TLS files.
func (c *Config) Validate() error {
	if len(c.Endpoints) == 0 {
		return errors.New("at least one endpoint must be configured")
	}
	if c.ClientConfig.Endpoint != "" {
		return errors.New("endpoint is unsupported; configure endpoints instead")
	}
	if c.ClientConfig.Timeout != 0 {
		return errors.New("timeout is unsupported; consumers own request deadlines")
	}
	if c.ClientConfig.Auth.HasValue() {
		return errors.New("nested auth is unsupported")
	}
	if len(c.ClientConfig.Middlewares) != 0 {
		return errors.New("middlewares are unsupported")
	}
	if c.ClientConfig.Cookies.HasValue() {
		return errors.New("cookies are unsupported")
	}
	if c.ClientConfig.Compression.IsCompressed() {
		return errors.New("compression is unsupported")
	}

	if c.ClientConfig.TLS.Insecure {
		return errors.New("tls.insecure is unsupported; use http:// endpoints or tls.insecure_skip_verify")
	}

	authHeaders := c.authorizationHeaderCount()
	if authHeaders > 1 {
		return errors.New("multiple Authorization headers are unsupported")
	}
	if (c.User != "" || c.Password != "" || c.APIKey != "") && authHeaders > 0 {
		return errors.New("an Authorization header cannot be combined with user, password, or api_key")
	}

	for _, endpoint := range c.Endpoints {
		u, err := parseHTTPURL(endpoint)
		if err != nil {
			return fmt.Errorf("invalid endpoint: %w", err)
		}
		if u.User != nil {
			return errors.New("invalid endpoint: endpoint userinfo is unsupported; configure user and password instead")
		}
	}

	if c.ClientConfig.ProxyURL != "" {
		if _, err := parseHTTPURL(c.ClientConfig.ProxyURL); err != nil {
			return fmt.Errorf("invalid proxy_url: %w", err)
		}
	}

	return c.validateAuthentication()
}

// authorizationHeaderCount returns the number of configured Authorization
// headers, compared case-insensitively.
func (c *Config) authorizationHeaderCount() int {
	count := 0
	for name := range c.ClientConfig.Headers.Iter {
		if strings.EqualFold(name, "Authorization") {
			count++
		}
	}
	return count
}

func (c *Config) validateAuthentication() error {
	hasUser := c.User != ""
	hasPassword := c.Password != ""
	if hasUser != hasPassword {
		return errors.New("user and password must be configured together")
	}
	if c.APIKey != "" && hasUser {
		return errors.New("api_key cannot be combined with basic authentication")
	}
	return nil
}

func parseHTTPURL(rawURL string) (*url.URL, error) {
	if rawURL == "" {
		return nil, errors.New("URL must not be empty")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, errors.New("URL must be a fully resolved HTTP(S) URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, errors.New("URL scheme must be http or https")
	}
	if u.Hostname() == "" {
		return nil, errors.New("URL host must not be empty")
	}
	if u.Fragment != "" {
		return nil, errors.New("URL fragments are unsupported")
	}
	return u, nil
}
