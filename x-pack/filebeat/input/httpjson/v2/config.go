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
}

func (c config) Validate() error {
	if c.Interval <= 0 {
		return errors.New("interval must be greater than 0")
	}
	return nil
}

func defaultConfig() config {
	timeout := 30 * time.Second
	return config{
		Interval: time.Minute,
		Auth:     &authConfig{},
		Request: &requestConfig{
			Timeout:                 &timeout,
			Method:                  "GET",
			RedirectHeadersForward:  true,
			RedirectLocationTrusted: false,
			RedirectHeadersBanList: []string{
				"WWW-Authenticate",
				"Authorization",
			},
			RedirectMaxRedirects: 10,
		},
		Response: &responseConfig{},
	}
}
