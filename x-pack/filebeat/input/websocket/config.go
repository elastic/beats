// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package websocket

import (
	"errors"
	"net/url"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

const defaultMaxExecutions = 1000

type config struct {
	// Program is the CEL program to be run for each polling.
	Program string `config:"program"`
	// MaxExecutions is the maximum number of times a single
	// periodic CEL execution loop may repeat due to a true
	// "want_more" field. If it is nil a sensible default is
	// used.
	MaxExecutions *int `config:"max_executions"`
	// Regexps is the set of regular expression to be made
	// available to the program.
	Regexps map[string]string `config:"regexp"`
	// State is the initial state to be provided to the
	// program. If it has a cursor field, that field will
	// be overwritten by any stored cursor, but will be
	// available if no stored cursor exists.
	State map[string]interface{} `config:"state"`
	// Auth is the authentication config for connection
	Auth authConfig `config:"auth"`
	// Resource
	Resource *ResourceConfig `config:"resource" validate:"required"`
}

type ResourceConfig struct {
	URL    *urlConfig         `config:"url" validate:"required"`
	Retry  retryConfig        `config:"retry"`
	Tracer *lumberjack.Logger `config:"tracer"`
}

type authConfig struct {
	// Api-Key to use for authentication.
	ApiKey *apiKeyConfig `config:"api_key"`
	// Baerer token to use for authentication.
	BearerToken string `config:"bearer_token"`
	// Basic auth token to use for authentication.
	BasicToken string `config:"basic_token"`
}

type apiKeyConfig struct {
	// Api-Key to use for authentication.
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

type retryConfig struct {
	MaxAttempts *int           `config:"max_attempts"`
	WaitMin     *time.Duration `config:"wait_min"`
	WaitMax     *time.Duration `config:"wait_max"`
}

func (c retryConfig) Validate() error {
	switch {
	case c.MaxAttempts != nil && *c.MaxAttempts <= 0:
		return errors.New("max_attempts must be greater than zero")
	case c.WaitMin != nil && *c.WaitMin <= 0:
		return errors.New("wait_min must be greater than zero")
	case c.WaitMax != nil && *c.WaitMax <= 0:
		return errors.New("wait_max must be greater than zero")
	}
	return nil
}

func defaultConfig() config {
	maxExecutions := defaultMaxExecutions
	maxAttempts := 5
	waitMin := time.Second
	waitMax := time.Minute

	return config{
		MaxExecutions: &maxExecutions,

		Resource: &ResourceConfig{
			Retry: retryConfig{
				MaxAttempts: &maxAttempts,
				WaitMin:     &waitMin,
				WaitMax:     &waitMax,
			},
		},
	}
}

func (c retryConfig) getMaxAttempts() int {
	if c.MaxAttempts == nil {
		return 0
	}
	return *c.MaxAttempts
}

func (c retryConfig) getWaitMin() time.Duration {
	if c.WaitMin == nil {
		return 0
	}
	return *c.WaitMin
}

func (c retryConfig) getWaitMax() time.Duration {
	if c.WaitMax == nil {
		return 0
	}
	return *c.WaitMax
}
