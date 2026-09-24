// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package elasticsearchauth provides Elasticsearch HTTP authentication for OTel components.
package elasticsearchauth

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/confmap"
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
	// APIKey configures Elasticsearch ApiKey authentication. It must be the
	// base64-encoded id:key representation accepted by Elasticsearch.
	APIKey configopaque.String `mapstructure:"api_key"`
}

var supportedConfigFields = map[string]struct{}{
	"api_key":                 {},
	"disable_keep_alives":     {},
	"endpoints":               {},
	"force_attempt_http2":     {},
	"headers":                 {},
	"http2_ping_timeout":      {},
	"http2_read_idle_timeout": {},
	"idle_conn_timeout":       {},
	"keepalive":               {},
	"max_conns_per_host":      {},
	"max_idle_conns":          {},
	"max_idle_conns_per_host": {},
	"password":                {},
	"proxy_url":               {},
	"read_buffer_size":        {},
	"tls":                     {},
	"user":                    {},
	"write_buffer_size":       {},
}

var unsupportedConfigFields = map[string]struct{}{
	"auth":               {},
	"compression":        {},
	"compression_params": {},
	"cookies":            {},
	"endpoint":           {},
	"middlewares":        {},
	"timeout":            {},
}

func createDefaultConfig() component.Config {
	return &Config{ClientConfig: confighttp.NewDefaultClientConfig()}
}

// Unmarshal preserves confighttp's custom nested unmarshaling while rejecting
// fields that this extension intentionally does not support.
func (c *Config) Unmarshal(conf *confmap.Conf) error {
	for _, key := range conf.AllKeys() {
		field, _, _ := strings.Cut(key, confmap.KeyDelimiter)
		if _, unsupported := unsupportedConfigFields[field]; unsupported {
			return unsupportedFieldError(field)
		}
		if _, known := supportedConfigFields[field]; !known {
			return fmt.Errorf("unsupported configuration key %q", field)
		}
	}

	decoded := configWithoutUnmarshal{ClientConfig: c.ClientConfig}
	if err := conf.Unmarshal(&decoded); err != nil {
		return err
	}

	c.ClientConfig = decoded.ClientConfig
	c.Endpoints = decoded.Endpoints
	c.User = decoded.User
	c.Password = decoded.Password
	c.APIKey = decoded.APIKey

	return nil
}

type configWithoutUnmarshal struct {
	ClientConfig confighttp.ClientConfig `mapstructure:",squash"`
	Endpoints    []string                `mapstructure:"endpoints"`
	User         string                  `mapstructure:"user"`
	Password     configopaque.String     `mapstructure:"password"`
	APIKey       configopaque.String     `mapstructure:"api_key"`
}

// Validate validates configuration relationships without accessing TLS files.
func (c *Config) Validate() error {
	if len(c.Endpoints) == 0 {
		return errors.New("at least one endpoint must be configured")
	}
	if c.ClientConfig.Endpoint != "" {
		return unsupportedFieldError("endpoint")
	}
	if c.ClientConfig.Timeout != 0 {
		return unsupportedFieldError("timeout")
	}
	if c.ClientConfig.Auth.HasValue() {
		return unsupportedFieldError("auth")
	}
	if len(c.ClientConfig.Middlewares) != 0 {
		return unsupportedFieldError("middlewares")
	}
	if c.ClientConfig.Cookies.HasValue() {
		return unsupportedFieldError("cookies")
	}
	if c.ClientConfig.Compression.IsCompressed() {
		return unsupportedFieldError("compression")
	}
	if err := c.ClientConfig.Headers.Validate(); err != nil {
		return fmt.Errorf("invalid headers: %w", err)
	}

	for _, endpoint := range c.Endpoints {
		u, err := parseHTTPURL(endpoint)
		if err != nil {
			return fmt.Errorf("invalid endpoint: %w", err)
		}
		if u.User != nil && (c.User != "" || c.Password != "" || c.APIKey != "") {
			return errors.New("endpoint userinfo cannot be combined with configured authentication")
		}
	}

	if c.ClientConfig.ProxyURL != "" {
		if _, err := parseHTTPURL(c.ClientConfig.ProxyURL); err != nil {
			return fmt.Errorf("invalid proxy_url: %w", err)
		}
	}

	if err := validateAuthentication(c.User, c.Password, c.APIKey); err != nil {
		return err
	}
	if err := c.ClientConfig.TLS.Validate(); err != nil {
		return fmt.Errorf("invalid TLS configuration: %w", err)
	}

	return c.ClientConfig.Validate()
}

func unsupportedFieldError(field string) error {
	switch field {
	case "endpoint":
		return errors.New("endpoint is unsupported; configure endpoints instead")
	case "timeout":
		return errors.New("timeout is unsupported; consumers own request deadlines")
	case "auth":
		return errors.New("nested auth is unsupported")
	case "compression_params":
		return errors.New("compression_params are unsupported")
	case "compression":
		return errors.New("compression is unsupported")
	default:
		return fmt.Errorf("%s are unsupported", field)
	}
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
	if u.Host == "" {
		return nil, errors.New("URL host must not be empty")
	}
	if u.Fragment != "" {
		return nil, errors.New("URL fragments are unsupported")
	}
	return u, nil
}

func validateAuthentication(user string, password, apiKey configopaque.String) error {
	hasUser := user != ""
	hasPassword := password != ""
	hasAPIKey := apiKey != ""
	if hasUser != hasPassword {
		return errors.New("user and password must be configured together")
	}
	if hasAPIKey && hasUser {
		return errors.New("api_key cannot be combined with basic authentication")
	}
	if !hasAPIKey {
		return nil
	}
	decoded, err := base64.StdEncoding.DecodeString(string(apiKey))
	if err != nil {
		return errors.New("api_key must be base64-encoded id:key")
	}
	id, key, found := strings.Cut(string(decoded), ":")
	if !found || id == "" || key == "" {
		return errors.New("api_key must be base64-encoded id:key")
	}
	return nil
}
