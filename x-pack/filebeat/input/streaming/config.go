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

	"github.com/elastic/elastic-agent-libs/logp"
)

type config struct {
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
	MaxAttempts int           `config:"max_attempts"`
	WaitMin     time.Duration `config:"wait_min"`
	WaitMax     time.Duration `config:"wait_max"`
}

type authConfig struct {
	// Custom auth config to use for authentication.
	CustomAuth *customAuthConfig `config:"custom"`
	// Baerer token to use for authentication.
	BearerToken string `config:"bearer_token"`
	// Basic auth token to use for authentication.
	BasicToken string `config:"basic_token"`
}

type customAuthConfig struct {
	// Custom auth config to use for authentication.
	Header string `config:"header"`
	Value  string `config:"value"`
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
	err = checkURLScheme(c.URL)
	if err != nil {
		return err
	}

	if c.Retry != nil {
		switch {
		case c.Retry.MaxAttempts <= 0:
			return errors.New("max_attempts must be greater than zero")
		case c.Retry.WaitMin > c.Retry.WaitMax:
			return errors.New("wait_min must be less than or equal to wait_max")
		}
	}
	return nil
}

func checkURLScheme(url *urlConfig) error {
	switch url.Scheme {
	case "ws", "wss":
		return nil
	default:
		return fmt.Errorf("unsupported scheme: %s", url.Scheme)
	}
}
