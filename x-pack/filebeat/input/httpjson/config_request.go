// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
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
	if c.WaitMax == nil && c.WaitMin == nil {
		return 0
	} else if c.WaitMax == nil && c.WaitMin != nil {
		return *c.WaitMin
	}
	return *c.WaitMax
}

type rateLimitConfig struct {
	Limit      *valueTpl `config:"limit"`
	Reset      *valueTpl `config:"reset"`
	Remaining  *valueTpl `config:"remaining"`
	EarlyLimit *float64  `config:"early_limit"`
}

func (c rateLimitConfig) Validate() error {
	switch {
	case c.EarlyLimit != nil && *c.EarlyLimit < 0:
		return errors.New("early_limit must be greater than or equal to 0")
	}
	return nil
}

type keepAlive struct {
	Disable             *bool         `config:"disable"`
	MaxIdleConns        int           `config:"max_idle_connections"`
	MaxIdleConnsPerHost int           `config:"max_idle_connections_per_host"` // If zero, http.DefaultMaxIdleConnsPerHost is the value used by http.Transport.
	IdleConnTimeout     time.Duration `config:"idle_connection_timeout"`
}

func (c keepAlive) Validate() error {
	if c.Disable == nil || *c.Disable {
		return nil
	}
	if c.MaxIdleConns < 0 {
		return errors.New("max_idle_connections must not be negative")
	}
	if c.MaxIdleConnsPerHost < 0 {
		return errors.New("max_idle_connections_per_host must not be negative")
	}
	if c.IdleConnTimeout < 0 {
		return errors.New("idle_connection_timeout must not be negative")
	}
	return nil
}

func (c keepAlive) settings() httpcommon.WithKeepaliveSettings {
	return httpcommon.WithKeepaliveSettings{
		Disable:             c.Disable == nil || *c.Disable,
		MaxIdleConns:        c.MaxIdleConns,
		MaxIdleConnsPerHost: c.MaxIdleConnsPerHost,
		IdleConnTimeout:     c.IdleConnTimeout,
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

	*u = urlConfig{URL: parsed}

	return nil
}

type requestConfig struct {
	URL                    *urlConfig       `config:"url" validate:"required"`
	Method                 string           `config:"method" validate:"required"`
	Body                   *mapstr.M        `config:"body"`
	EncodeAs               string           `config:"encode_as"`
	Retry                  retryConfig      `config:"retry"`
	RedirectForwardHeaders bool             `config:"redirect.forward_headers"`
	RedirectHeadersBanList []string         `config:"redirect.headers_ban_list"`
	RedirectMaxRedirects   int              `config:"redirect.max_redirects"`
	RateLimit              *rateLimitConfig `config:"rate_limit"`
	KeepAlive              keepAlive        `config:"keep_alive"`
	Transforms             transformsConfig `config:"transforms"`

	Transport httpcommon.HTTPTransportSettings `config:",inline"`

	Tracer *tracerConfig `config:"tracer"`
}

type tracerConfig struct {
	Enabled           *bool `config:"enabled"`
	lumberjack.Logger `config:",inline"`
}

func (t *tracerConfig) enabled() bool {
	return t != nil && (t.Enabled == nil || *t.Enabled)
}

func (c *requestConfig) Validate() error {
	c.Method = strings.ToUpper(c.Method)
	switch c.Method {
	case http.MethodPost:
	case http.MethodGet:
		if c.Body != nil {
			return errors.New("body can't be used with method: \"GET\"")
		}
	default:
		return fmt.Errorf("unsupported method %q", c.Method)
	}

	if _, err := newBasicTransformsFromConfig(registeredTransforms, c.Transforms, requestNamespace, nil); err != nil {
		return err
	}

	if c.EncodeAs != "" {
		if _, found := registeredEncoders[c.EncodeAs]; !found {
			return fmt.Errorf("encoder not found for contentType: %v", c.EncodeAs)
		}
	}

	if c.Tracer != nil {
		if c.Tracer.Filename == "" {
			return errors.New("request tracer must have a filename if used")
		}
		if c.Tracer.MaxSize == 0 {
			// By default Lumberjack caps file sizes at 100MB which
			// is excessive for a debugging logger, so default to 1MB
			// which is the minimum.
			c.Tracer.MaxSize = 1
		}
	}

	return nil
}
