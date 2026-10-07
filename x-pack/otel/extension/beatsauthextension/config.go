// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatsauthextension

import (
	"errors"
	"fmt"
	"net/url"

	"go.opentelemetry.io/collector/component"

	"github.com/elastic/beats/v7/libbeat/common/transport/kerberos"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

type Config struct {
	BeatAuthConfig  map[string]any `mapstructure:",remain"`
	ContinueOnError bool           `mapstructure:"continue_on_error"`

	// Endpoints contains optional, fully resolved Elasticsearch HTTP(S)
	// endpoints.
	Endpoints []string `mapstructure:"endpoints,omitempty"`
}

type BeatsAuthConfig struct {
	Kerberos  *kerberos.Config                 `config:"kerberos"`
	Transport httpcommon.HTTPTransportSettings `config:",inline"`
}

func createDefaultConfig() component.Config {
	return &Config{}
}

// Validate validates optional Elasticsearch endpoints.
func (c *Config) Validate() error {
	for _, endpoint := range c.Endpoints {
		u, err := parseHTTPURL(endpoint)
		if err != nil {
			return fmt.Errorf("invalid endpoint: %w", err)
		}
		if u.User != nil {
			return errors.New("endpoint userinfo is unsupported; configure auth.username and auth.password instead")
		}
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
	if u.Host == "" {
		return nil, errors.New("URL host must not be empty")
	}
	if u.Fragment != "" {
		return nil, errors.New("URL fragments are unsupported")
	}
	return u, nil
}
