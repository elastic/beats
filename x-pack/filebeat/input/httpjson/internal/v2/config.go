// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"errors"
	"time"
)

type config struct {
	Interval time.Duration   `config:"interval" validate:"required"`
	Auth     *authConfig     `config:"auth"`
	Request  *requestConfig  `config:"request" validate:"required"`
	Response *responseConfig `config:"response"`
	Cursor   cursorConfig    `config:"cursor"`
}

type cursorConfig map[string]cursorEntry

type cursorEntry struct {
	Value            *valueTpl `config:"value"`
	Default          *valueTpl `config:"default"`
	IgnoreEmptyValue *bool     `config:"ignore_empty_value"`
}

func (ce cursorEntry) mustIgnoreEmptyValue() bool {
	return ce.IgnoreEmptyValue == nil || *ce.IgnoreEmptyValue
}

func (c config) Validate() error {
	if c.Interval <= 0 {
		return errors.New("interval must be greater than 0")
	}
	return nil
}

func defaultConfig() config {
	timeout := 30 * time.Second
	maxAttempts := 5
	waitMin := time.Second
	waitMax := time.Minute
	return config{
		Interval: time.Minute,
		Auth:     &authConfig{},
		Request: &requestConfig{
			Timeout: &timeout,
			Method:  "GET",
			Retry: retryConfig{
				MaxAttempts: &maxAttempts,
				WaitMin:     &waitMin,
				WaitMax:     &waitMax,
			},
			RedirectForwardHeaders: false,
			RedirectMaxRedirects:   10,
		},
		Response: &responseConfig{},
	}
}
