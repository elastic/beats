// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package okta

import (
	"errors"
	"strings"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

// defaultConfig returns a default configuration.
func defaultConfig() conf {
	maxAttempts := 5
	waitMin := time.Second
	waitMax := time.Minute
	transport := httpcommon.DefaultHTTPTransportSettings()
	transport.Timeout = 30 * time.Second

	return conf{
		EnrichWith:     []string{"groups"},
		SyncInterval:   24 * time.Hour,
		UpdateInterval: 15 * time.Minute,
		LimitWindow:    time.Minute,
		Request: &requestConfig{
			Retry: retryConfig{
				MaxAttempts: &maxAttempts,
				WaitMin:     &waitMin,
				WaitMax:     &waitMax,
			},
			RedirectForwardHeaders: false,
			RedirectMaxRedirects:   10,
			Transport:              transport,
		},
	}
}

// conf contains parameters needed to configure the input.
type conf struct {
	OktaDomain string `config:"okta_domain" validate:"required"`
	OktaToken  string `config:"okta_token" validate:"required"`

	// Dataset specifies the datasets to collect from
	// the API. It can be ""/"all", "users", or
	// "devices".
	Dataset string `config:"dataset"`
	// EnrichWith specifies the additional data that
	// will be used to enrich user data. It can include
	// "groups", "roles" and "factors".
	// If it is a single element with "none", no
	// enrichment is performed.
	EnrichWith []string `config:"enrich_with"`

	// SyncInterval is the time between full
	// synchronisation operations.
	SyncInterval time.Duration `config:"sync_interval"`

	// UpdateInterval is the time between
	// incremental updated.
	UpdateInterval time.Duration `config:"update_interval"`

	// LimitWindow is the time between Okta
	// API limit resets.
	LimitWindow time.Duration `config:"limit_window"`

	// Request is the configuration for establishing
	// HTTP requests to the API.
	Request *requestConfig `config:"request"`

	// Tracer allows configuration of request trace logging.
	Tracer *tracerConfig `config:"tracer"`
}

type tracerConfig struct {
	Enabled           *bool `config:"enabled"`
	lumberjack.Logger `config:",inline"`
}

func (t *tracerConfig) enabled() bool {
	return t != nil && (t.Enabled == nil || *t.Enabled)
}

type requestConfig struct {
	Retry                  retryConfig `config:"retry"`
	RedirectForwardHeaders bool        `config:"redirect.forward_headers"`
	RedirectHeadersBanList []string    `config:"redirect.headers_ban_list"`
	RedirectMaxRedirects   int         `config:"redirect.max_redirects"`
	KeepAlive              keepAlive   `config:"keep_alive"`

	Transport httpcommon.HTTPTransportSettings `config:",inline"`
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

var (
	errInvalidSyncInterval   = errors.New("zero or negative sync_interval")
	errInvalidUpdateInterval = errors.New("zero or negative update_interval")
	errSyncBeforeUpdate      = errors.New("sync_interval not longer than update_interval")
)

// Validate runs validation against the config.
func (c *conf) Validate() error {
	switch {
	case c.SyncInterval <= 0:
		return errInvalidSyncInterval
	case c.UpdateInterval <= 0:
		return errInvalidUpdateInterval
	case c.SyncInterval <= c.UpdateInterval:
		return errSyncBeforeUpdate
	}
	switch strings.ToLower(c.Dataset) {
	case "", "all", "users", "devices":
	default:
		return errors.New("dataset must be 'all', 'users', 'devices' or empty")
	}

	if c.Tracer == nil {
		return nil
	}
	if c.Tracer.Filename == "" {
		return errors.New("request tracer must have a filename if used")
	}
	if c.Tracer.MaxSize == 0 {
		// By default Lumberjack caps file sizes at 100MB which
		// is excessive for a debugging logger, so default to 1MB
		// which is the minimum.
		c.Tracer.MaxSize = 1
	}
	return nil
}

func (c *conf) wantUsers() bool {
	switch strings.ToLower(c.Dataset) {
	case "", "all", "users":
		return true
	default:
		return false
	}
}

func (c *conf) wantDevices() bool {
	switch strings.ToLower(c.Dataset) {
	case "", "all", "devices":
		return true
	default:
		return false
	}
}
