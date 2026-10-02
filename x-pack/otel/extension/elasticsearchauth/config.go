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
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
)

// Config configures the Elasticsearch authentication extension.
type Config struct {
	// Auth optionally identifies an HTTP client authenticator that owns the
	// underlying transport and all transport-level configuration.
	Auth configoptional.Optional[configauth.Config] `mapstructure:"auth,omitempty"`

	// Endpoints contains the resolved Elasticsearch HTTP(S) endpoints.
	Endpoints []string `mapstructure:"endpoints"`

	// Headers contains destination headers applied to Elasticsearch requests.
	Headers configopaque.MapList `mapstructure:"headers,omitempty"`

	// User configures HTTP Basic authentication with Password.
	User string `mapstructure:"user"`
	// Password configures HTTP Basic authentication with User.
	Password configopaque.String `mapstructure:"password"`
	// APIKey configures Elasticsearch ApiKey authentication as a raw, non-empty
	// id:key pair. The extension encodes it when constructing the request header.
	APIKey configopaque.String `mapstructure:"api_key"`
}

func createDefaultConfig() component.Config {
	return &Config{Auth: configoptional.None[configauth.Config]()}
}

// Validate validates Elasticsearch destination configuration.
func (c *Config) Validate() error {
	if len(c.Endpoints) == 0 {
		return errors.New("at least one endpoint must be configured")
	}

	hasExplicitCredentials := c.User != "" || c.Password != "" || c.APIKey != ""
	hasAuthorizationHeader := hasHeader(c.Headers, "Authorization")
	if hasExplicitCredentials && hasAuthorizationHeader {
		return errors.New("authorization header cannot be combined with user, password, or api_key")
	}

	for _, endpoint := range c.Endpoints {
		u, err := parseHTTPURL(endpoint)
		if err != nil {
			return fmt.Errorf("invalid endpoint: %w", err)
		}

		if u.User != nil && (hasExplicitCredentials || hasAuthorizationHeader) {
			return errors.New("endpoint userinfo cannot be combined with configured authentication")
		}
	}

	if err := validateAuthentication(c.User, c.Password, c.APIKey); err != nil {
		return err
	}

	return nil
}

func hasHeader(headers configopaque.MapList, name string) bool {
	for header := range headers.Iter {
		if strings.EqualFold(header, name) {
			return true
		}
	}
	return false
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

	id, key, found := strings.Cut(string(apiKey), ":")
	if !found || id == "" || key == "" {
		return errors.New("api_key must be raw non-empty id:key")
	}

	return nil
}
