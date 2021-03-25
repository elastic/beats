// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
)

type retryConfig struct {
	MaxAttempts *int           `config:"max_attempts"`
	WaitMin     *time.Duration `config:"wait_min"`
	WaitMax     *time.Duration `config:"wait_max"`
}

func (c retryConfig) Validate() error {
	switch {
	case c.MaxAttempts != nil && *c.MaxAttempts <= 0:
		return errors.New("max_attempts must be greater than 0")
	case c.WaitMin != nil && *c.WaitMin <= 0:
		return errors.New("wait_min must be greater than 0")
	case c.WaitMax != nil && *c.WaitMax <= 0:
		return errors.New("wait_max must be greater than 0")
	}
	return nil
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

type rateLimitConfig struct {
	Limit     *valueTpl `config:"limit"`
	Reset     *valueTpl `config:"reset"`
	Remaining *valueTpl `config:"remaining"`
}

type urlConfig struct {
	*url.URL
}

func (u *urlConfig) Unpack(in string) error {
	parsed, err := url.Parse(in)
	if err != nil {
		return err
	}

	*u = urlConfig{URL: parsed}

	return nil
}

type requestConfig struct {
	URL                    *urlConfig        `config:"url" validate:"required"`
	Method                 string            `config:"method" validate:"required"`
	Body                   *common.MapStr    `config:"body"`
	EncodeAs               string            `config:"encode_as"`
	Timeout                *time.Duration    `config:"timeout"`
	SSL                    *tlscommon.Config `config:"ssl"`
	Retry                  retryConfig       `config:"retry"`
	RedirectForwardHeaders bool              `config:"redirect.forward_headers"`
	RedirectHeadersBanList []string          `config:"redirect.headers_ban_list"`
	RedirectMaxRedirects   int               `config:"redirect.max_redirects"`
	RateLimit              *rateLimitConfig  `config:"rate_limit"`
	Transforms             transformsConfig  `config:"transforms"`
	ProxyURL               *urlConfig        `config:"proxy_url"`
}

func (c requestConfig) getTimeout() time.Duration {
	if c.Timeout == nil {
		return 0
	}
	return *c.Timeout
}

func (c *requestConfig) Validate() error {
	c.Method = strings.ToUpper(c.Method)
	switch c.Method {
	case "POST":
	case "GET":
		if c.Body != nil {
			return errors.New("body can't be used with method: \"GET\"")
		}
	default:
		return fmt.Errorf("unsupported method %q", c.Method)
	}

	if c.Timeout != nil && *c.Timeout <= 0 {
		return errors.New("timeout must be greater than 0")
	}

	if _, err := newBasicTransformsFromConfig(c.Transforms, requestNamespace, nil); err != nil {
		return err
	}

	if c.EncodeAs != "" {
		if _, found := registeredEncoders[c.EncodeAs]; !found {
			return fmt.Errorf("encoder not found for contentType: %v", c.EncodeAs)
		}
	}

	return nil
}
