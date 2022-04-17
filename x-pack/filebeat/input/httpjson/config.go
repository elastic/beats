// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"errors"
	"time"

	"github.com/menderesk/beats/v7/libbeat/common/transport/httpcommon"
)

type config struct {
	Interval time.Duration   `config:"interval" validate:"required"`
	Auth     *authConfig     `config:"auth"`
	Request  *requestConfig  `config:"request" validate:"required"`
	Response *responseConfig `config:"response"`
	Cursor   cursorConfig    `config:"cursor"`
	Chain    []chainConfig   `config:"chain"`
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
	maxAttempts := 5
	waitMin := time.Second
	waitMax := time.Minute
	transport := httpcommon.DefaultHTTPTransportSettings()
	transport.Timeout = 30 * time.Second

	return config{
		Interval: time.Minute,
		Auth:     &authConfig{},
		Request: &requestConfig{
			Method: "GET",
			Retry: retryConfig{
				MaxAttempts: &maxAttempts,
				WaitMin:     &waitMin,
				WaitMax:     &waitMax,
			},
			RedirectForwardHeaders: false,
			RedirectMaxRedirects:   10,
			Transport:              transport,
		},
		Response: &responseConfig{},
	}
}
