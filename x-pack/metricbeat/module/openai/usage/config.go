// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package usage

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

type Config struct {
	APIKeys    []apiKeyConfig   `config:"api_keys" validate:"required"`
	APIURL     string           `config:"api_url" validate:"required"`
	Headers    []string         `config:"headers"`
	RateLimit  *rateLimitConfig `config:"rate_limit"`
	Timeout    time.Duration    `config:"timeout" validate:"required"`
	Collection collectionConfig `config:"collection"`
}

type rateLimitConfig struct {
	Limit *int `config:"limit" validate:"required"`
	Burst *int `config:"burst" validate:"required"`
}

type apiKeyConfig struct {
	Key string `config:"key" validate:"required"`
}

type collectionConfig struct {
	LookbackDays int  `config:"lookback_days"`
	Realtime     bool `config:"realtime"`
}

func defaultConfig() Config {
	return Config{
		APIURL:  "https://api.openai.com/v1/usage",
		Timeout: 30 * time.Second,
		RateLimit: &rateLimitConfig{
			Limit: ptr(12),
			Burst: ptr(1),
		},
		Collection: collectionConfig{
			LookbackDays: 0,     // 0 days
			Realtime:     false, // avoid realtime collection by default
		},
	}
}

func (c *Config) Validate() error {
	var errs []error

	if len(c.APIKeys) == 0 {
		errs = append(errs, errors.New("at least one API key must be configured"))
	}
	if c.APIURL == "" {
		errs = append(errs, errors.New("api_url cannot be empty"))
	} else {
		_, err := url.ParseRequestURI(c.APIURL)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid api_url format: %w", err))
		}
	}
	if c.RateLimit == nil {
		errs = append(errs, errors.New("rate_limit must be configured"))
	} else {
		if c.RateLimit.Limit == nil {
			errs = append(errs, errors.New("rate_limit.limit must be configured"))
		}
		if c.RateLimit.Burst == nil {
			errs = append(errs, errors.New("rate_limit.burst must be configured"))
		}
	}
	if c.Timeout <= 0 {
		errs = append(errs, errors.New("timeout must be greater than 0"))
	}
	if c.Collection.LookbackDays < 0 {
		errs = append(errs, errors.New("lookback_days must be >= 0"))
	}

	for i, apiKey := range c.APIKeys {
		if apiKey.Key == "" {
			errs = append(errs, fmt.Errorf("API key at position %d cannot be empty", i))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed: %w", errors.Join(errs...))
	}

	return nil
}
