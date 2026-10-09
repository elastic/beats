// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatsauthextension

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"go.opentelemetry.io/collector/component"

	"github.com/elastic/beats/v7/libbeat/common/transport/kerberos"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

type Config struct {
	// BeatAuthConfig contains the unmatched extension settings decoded by the
	// Collector. They must be decoded again using the Beats config parser.
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

// Validate validates optional Elasticsearch endpoints and authentication.
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

	// The Collector does not unmarhsal BeatAuthConfig so
	// we unmarshal and validate it.
	beatsAuthCfg, _, err := c.beatsAuthConfig()
	if err != nil {
		return err
	}

	return beatsAuthCfg.validate()
}

func (c *Config) beatsAuthConfig() (BeatsAuthConfig, *config.C, error) {
	// The Collector preserves transport settings in BeatAuthConfig because
	// Config does not declare the Beats-specific auth and transport fields.
	parsedCfg, err := config.NewConfigFrom(c.BeatAuthConfig)
	if err != nil {
		return BeatsAuthConfig{}, nil, fmt.Errorf("failed creating config: %w", err)
	}

	beatAuthConfig := BeatsAuthConfig{}
	if err := parsedCfg.Unpack(&beatAuthConfig); err != nil {
		return BeatsAuthConfig{}, nil, fmt.Errorf("failed unpacking config: %w", err)
	}

	return beatAuthConfig, parsedCfg, nil
}

func (c *BeatsAuthConfig) validate() error {
	auth := c.Transport.Auth
	authorizationMethods := 0
	if auth != nil {
		if auth.APIKey != "" {
			authorizationMethods++
		}
		if auth.Username != "" || auth.Password != "" {
			if auth.APIKey != "" {
				return errors.New("cannot set both api_key and username/password")
			}
			authorizationMethods++
		}

		authorizationHeaders := 0
		for _, header := range auth.Headers {
			if strings.EqualFold(header.Key, "Authorization") {
				authorizationHeaders++
			}
		}
		if authorizationHeaders > 1 {
			return errors.New("cannot configure multiple Authorization headers")
		}
		authorizationMethods += authorizationHeaders
	}

	if authorizationMethods > 1 {
		return errors.New("cannot configure multiple HTTP authorization methods")
	}

	if c.Kerberos.IsEnabled() && authorizationMethods > 0 {
		return errors.New("cannot combine Kerberos with HTTP authorization")
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
