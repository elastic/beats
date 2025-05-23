// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"golang.org/x/oauth2"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

const (
	authStyleInHeader = "in_header"
	authStyleInParams = "in_params"
)

type config struct {
	// Type is the type of the stream being followed. The
	// zero value indicates websocket.
	Type string `config:"stream_type"`

	// URLProgram is the CEL program to be run once before to prep the url.
	URLProgram string `config:"url_program"`
	// Program is the CEL program to be run for each polling.
	Program string `config:"program"`
	// Regexps is the set of regular expression to be made
	// available to the program.
	Regexps map[string]string `config:"regexp"`
	// State is the initial state to be provided to the
	// program. If it has a cursor field, that field will
	// be overwritten by any stored cursor, but will be
	// available if no stored cursor exists.
	State map[string]interface{} `config:"state"`
	// Auth is the authentication config for connection.
	Auth authConfig `config:"auth"`
	// URL is the websocket url to connect to.
	URL *urlConfig `config:"url" validate:"required"`
	// Redact is the debug log state redaction configuration.
	Redact *redact `config:"redact"`
	// Retry is the configuration for retrying failed connections.
	Retry *retry `config:"retry"`
	// Transport is the common the transport config.
	Transport httpcommon.HTTPTransportSettings `config:",inline"`
	// KeepAlive is the configuration for keep-alive settings.
	KeepAlive keepAliveConfig `config:"keep_alive"`
	// CrowdstrikeAppID is the value used to set the
	// appId request parameter in the FalconHose stream
	// discovery request.
	CrowdstrikeAppID string `config:"crowdstrike_app_id"`
}

type redact struct {
	// Fields indicates which fields to apply redaction to prior
	// to logging.
	Fields []string `config:"fields"`
	// Delete indicates that fields should be completely deleted
	// before logging rather than redaction with a "*".
	Delete bool `config:"delete"`
}

type retry struct {
	MaxAttempts     int           `config:"max_attempts"`
	WaitMin         time.Duration `config:"wait_min"`
	WaitMax         time.Duration `config:"wait_max"`
	BlanketRetries  bool          `config:"blanket_retries"`
	InfiniteRetries bool          `config:"infinite_retries"`
}

// keepAliveConfig is the configuration for keep-alive settings.
type keepAliveConfig struct {
	// Enable indicates whether keep-alive is enabled.
	Enable bool `config:"enable"`
	// Interval is the interval between keep-alive messages.
	// by default, this is set to 30 seconds.
	Interval time.Duration `config:"interval"`
	// WriteControlDeadline is the deadline for write control messages.
	// by default, this is set to 10 seconds.
	WriteControlDeadline time.Duration `config:"write_control_deadline"`
	// readControlDeadline is the deadline for read control messages.
	// this is internally set to 3x the value of the writeControlDeadline to account for network latency/jitter.
	readControlDeadline time.Duration
}
type authConfig struct {
	// Custom auth config to use for authentication.
	CustomAuth *customAuthConfig `config:"custom"`
	// Baerer token to use for authentication.
	BearerToken string `config:"bearer_token"`
	// Basic auth token to use for authentication.
	BasicToken string `config:"basic_token"`

	OAuth2 oAuth2Config `config:",inline"`
}

type customAuthConfig struct {
	// Custom auth config to use for authentication.
	Header string `config:"header"`
	Value  string `config:"value"`
}

type oAuth2Config struct {
	// common oauth fields
	AuthStyle         string        `config:"auth_style"`
	ClientID          string        `config:"client_id"`
	ClientSecret      string        `config:"client_secret"`
	EndpointParams    url.Values    `config:"endpoint_params"`
	Scopes            []string      `config:"scopes"`
	TokenExpiryBuffer time.Duration `config:"token_expiry_buffer" validate:"min=0"`
	TokenURL          string        `config:"token_url"`
	// accessToken is only used internally to set the initial headers via formHeader() if oauth2 is enabled
	accessToken string
}

func (o oAuth2Config) isEnabled() bool {
	return o.ClientID != "" && o.ClientSecret != "" && o.TokenURL != ""
}

func (o oAuth2Config) getAuthStyle() oauth2.AuthStyle {
	switch o.AuthStyle {
	case authStyleInHeader:
		return oauth2.AuthStyleInHeader
	case authStyleInParams:
		return oauth2.AuthStyleInParams
	default:
		return oauth2.AuthStyleAutoDetect
	}
}

type urlConfig struct {
	*url.URL
}

func (u *urlConfig) Unpack(in string) error {
	parsed, err := url.Parse(in)
	if err != nil {
		return err
	}
	u.URL = parsed
	return nil
}

func (c config) Validate() error {
	switch c.Type {
	case "", "websocket", "crowdstrike":
	default:
		return fmt.Errorf("unknown stream type: %s", c.Type)
	}

	if c.Redact == nil {
		logp.L().Named("input.websocket").Warn("missing recommended 'redact' configuration: " +
			"see documentation for details: https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-websocket.html#_redact")
	}
	_, err := regexpsFromConfig(c)
	if err != nil {
		return fmt.Errorf("failed to check regular expressions: %w", err)
	}

	var patterns map[string]*regexp.Regexp
	if len(c.Regexps) != 0 {
		patterns = map[string]*regexp.Regexp{".": nil}
	}
	if c.Program != "" {
		_, _, err = newProgram(context.Background(), c.Program, root, patterns, logp.L().Named("input.websocket"))
		if err != nil {
			return fmt.Errorf("failed to check program: %w", err)
		}
	}
	err = checkURLScheme(c)
	if err != nil {
		return err
	}

	if c.Retry != nil {
		switch {
		case c.Retry.MaxAttempts <= 0 && !c.Retry.InfiniteRetries:
			return errors.New("max_attempts must be greater than zero")
		case c.Retry.WaitMin > c.Retry.WaitMax:
			return errors.New("wait_min must be less than or equal to wait_max")
		}
	}

	if c.Auth.OAuth2.isEnabled() {
		if c.Auth.OAuth2.AuthStyle != authStyleInHeader && c.Auth.OAuth2.AuthStyle != authStyleInParams && c.Auth.OAuth2.AuthStyle != "" {
			return fmt.Errorf("unsupported auth style: %s", c.Auth.OAuth2.AuthStyle)
		}
	}
	return nil
}

func checkURLScheme(c config) error {
	switch c.Type {
	case "", "websocket":
		switch c.URL.Scheme {
		case "ws", "wss":
			return nil
		default:
			return fmt.Errorf("unsupported scheme: %s", c.URL.Scheme)
		}
	case "crowdstrike":
		switch c.URL.Scheme {
		case "http", "https":
			return nil
		default:
			return fmt.Errorf("unsupported scheme: %s", c.URL.Scheme)
		}
	default:
		return fmt.Errorf("unknown stream type: %s", c.Type)
	}
}

func defaultConfig() config {
	return config{
		Transport: httpcommon.HTTPTransportSettings{
			Timeout: 180 * time.Second,
		},
		Auth: authConfig{
			OAuth2: oAuth2Config{
				TokenExpiryBuffer: 2 * time.Minute,
			},
		},
		Retry: &retry{
			MaxAttempts: 5,
			WaitMin:     1 * time.Second,
			WaitMax:     30 * time.Second,
		},
		KeepAlive: keepAliveConfig{
			Interval:             30 * time.Second,
			WriteControlDeadline: 10 * time.Second,
		},
	}
}
