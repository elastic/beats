// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package akamai

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

const (
	defaultInterval         = time.Minute
	defaultInitialInterval  = 12 * time.Hour
	defaultRecoveryInterval = 1 * time.Hour
	defaultEventLimit       = 10000
	maxEventLimit           = 600000
	defaultNumberOfWorkers  = 3
	defaultMaxAttempts      = 5
	defaultWaitMin          = time.Second
	defaultWaitMax          = time.Minute
	defaultInvalidTSRetries = 2
	maxInitialInterval      = 12 * time.Hour
)

// config is the top-level configuration for the akamai input.
type config struct {
	// Resource contains HTTP resource, transport and retry configuration.
	Resource *resourceConfig `config:"resource" validate:"required"`

	// ConfigIDs is a semicolon-separated list of security configuration IDs to monitor.
	ConfigIDs string `config:"config_ids" validate:"required"`

	// Auth contains the Akamai EdgeGrid authentication credentials.
	Auth authConfig `config:"auth"`

	// Interval is the polling interval for API requests.
	Interval time.Duration `config:"interval"`

	// InitialInterval is the lookback period for the first poll.
	// Maximum is 12 hours as per Akamai API limits.
	InitialInterval time.Duration `config:"initial_interval"`

	// RecoveryInterval is the lookback period when in recovery mode.
	// Maximum is 12 hours as per Akamai API limits.
	RecoveryInterval time.Duration `config:"recovery_interval"`

	// EventLimit is the maximum number of events per request.
	// Default is 10000, maximum is 600000.
	EventLimit int `config:"event_limit"`

	// NumberOfWorkers is the number of concurrent workers for processing events.
	NumberOfWorkers int `config:"number_of_workers"`

	// InvalidTimestampRetries is the number of immediate retries for 400
	// responses containing "invalid timestamp" before failing the request and entering recovery mode.
	InvalidTimestampRetries int `config:"invalid_timestamp_retry.max_attempts"`

	// Tracer configures request/response tracing for debugging.
	Tracer *tracerConfig `config:"tracer"`
}

// resourceConfig contains HTTP transport and retry configuration.
type resourceConfig struct {
	URL       *urlConfig                       `config:"url" validate:"required"`
	Retry     retryConfig                      `config:"retry"`
	Timeout   time.Duration                    `config:"timeout"`
	Transport httpcommon.HTTPTransportSettings `config:",inline"`
	KeepAlive keepAliveConfig                  `config:"keep_alive"`
	RateLimit *rateLimitConfig                 `config:"rate_limit"`
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

func (c retryConfig) getMaxAttempts() int {
	if c.MaxAttempts == nil {
		return defaultMaxAttempts
	}
	return *c.MaxAttempts
}

func (c retryConfig) getWaitMin() time.Duration {
	if c.WaitMin == nil {
		return defaultWaitMin
	}
	return *c.WaitMin
}

func (c retryConfig) getWaitMax() time.Duration {
	if c.WaitMax == nil {
		return defaultWaitMax
	}
	return *c.WaitMax
}

type rateLimitConfig struct {
	Limit *float64 `config:"limit"`
	Burst *int     `config:"burst"`
}

func (c rateLimitConfig) Validate() error {
	if c.Limit != nil && *c.Limit <= 0 {
		return errors.New("limit must be greater than zero")
	}
	if c.Limit == nil && c.Burst != nil && *c.Burst <= 0 {
		return errors.New("burst must be greater than zero if limit is not specified")
	}
	return nil
}

type keepAliveConfig struct {
	Disable             *bool         `config:"disable"`
	MaxIdleConns        int           `config:"max_idle_connections"`
	MaxIdleConnsPerHost int           `config:"max_idle_connections_per_host"`
	IdleConnTimeout     time.Duration `config:"idle_connection_timeout"`
}

func (c keepAliveConfig) Validate() error {
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

func (c keepAliveConfig) settings() httpcommon.WithKeepaliveSettings {
	return httpcommon.WithKeepaliveSettings{
		Disable:             c.Disable == nil || *c.Disable,
		MaxIdleConns:        c.MaxIdleConns,
		MaxIdleConnsPerHost: c.MaxIdleConnsPerHost,
		IdleConnTimeout:     c.IdleConnTimeout,
	}
}

type tracerConfig struct {
	Enabled           *bool `config:"enabled"`
	lumberjack.Logger `config:",inline"`
}

func (t *tracerConfig) enabled() bool {
	return t != nil && (t.Enabled == nil || *t.Enabled)
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

func defaultConfig() config {
	maxAttempts := defaultMaxAttempts
	waitMin := defaultWaitMin
	waitMax := defaultWaitMax
	transport := httpcommon.DefaultHTTPTransportSettings()
	transport.Timeout = 60 * time.Second

	return config{
		Interval:                defaultInterval,
		InitialInterval:         defaultInitialInterval,
		RecoveryInterval:        defaultRecoveryInterval,
		EventLimit:              defaultEventLimit,
		NumberOfWorkers:         defaultNumberOfWorkers,
		InvalidTimestampRetries: defaultInvalidTSRetries,
		Resource: &resourceConfig{
			Retry: retryConfig{
				MaxAttempts: &maxAttempts,
				WaitMin:     &waitMin,
				WaitMax:     &waitMax,
			},
			Timeout:   60 * time.Second,
			Transport: transport,
		},
	}
}

func (c *config) Validate() error {
	if c.Resource == nil || c.Resource.URL == nil || c.Resource.URL.URL == nil {
		return errors.New("resource.url is required")
	}
	if c.Resource.URL.Scheme != "https" && c.Resource.URL.Scheme != "http" {
		return errors.New("resource.url must use http or https scheme")
	}

	if c.ConfigIDs == "" {
		return errors.New("config_ids is required")
	}

	if err := c.Auth.Validate(); err != nil {
		return err
	}

	if c.Interval <= 0 {
		return errors.New("interval must be greater than 0")
	}

	if c.InitialInterval <= 0 {
		return errors.New("initial_interval must be greater than 0")
	}
	if c.InitialInterval > maxInitialInterval {
		return fmt.Errorf("initial_interval cannot exceed %v (Akamai API limit)", maxInitialInterval)
	}

	if c.RecoveryInterval <= 0 {
		return errors.New("recovery_interval must be greater than 0")
	}
	if c.RecoveryInterval > maxInitialInterval {
		return fmt.Errorf("recovery_interval cannot exceed %v (Akamai API limit)", maxInitialInterval)
	}

	if c.EventLimit <= 0 {
		return errors.New("event_limit must be greater than 0")
	}
	if c.EventLimit > maxEventLimit {
		return fmt.Errorf("event_limit cannot exceed %d", maxEventLimit)
	}

	if c.NumberOfWorkers <= 0 {
		return errors.New("number_of_workers must be greater than 0")
	}
	if c.InvalidTimestampRetries < 0 {
		return errors.New("invalid_timestamp_retry.max_attempts must be greater than or equal to 0")
	}

	if c.Tracer != nil && c.Tracer.enabled() && c.Tracer.Filename == "" {
		return errors.New("tracer filename is required when tracer is enabled")
	}

	return nil
}
