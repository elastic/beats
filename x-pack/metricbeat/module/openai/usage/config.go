// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package usage

import (
	"fmt"
	"time"
)

type Config struct {
	APIKeys    []apiKeyConfig   `config:"api_keys" validate:"required"`
	APIURL     string           `config:"api_url"`
	Headers    []string         `config:"headers"`
	RateLimit  *rateLimitConfig `config:"rate_limit"`
	Timeout    time.Duration    `config:"timeout"`
	Collection collectionConfig `config:"collection"`
}

type rateLimitConfig struct {
	Limit *int `config:"limit"`
	Burst *int `config:"burst"`
}

type apiKeyConfig struct {
	Key string `config:"key"`
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
			Limit: ptr(60),
			Burst: ptr(5),
		},
		Collection: collectionConfig{
			LookbackDays: 0,     // 0 days
			Realtime:     false, // avoid realtime collection by default
		},
	}
}

func (c *Config) Validate() error {
	switch {
	case len(c.APIKeys) == 0:
		return fmt.Errorf("at least one API key must be configured")

	case c.APIURL == "":
		return fmt.Errorf("api_url cannot be empty")

	case c.RateLimit == nil:
		return fmt.Errorf("rate_limit must be configured")

	case c.RateLimit.Limit == nil:
		return fmt.Errorf("rate_limit.limit must be configured")

	case c.RateLimit.Burst == nil:
		return fmt.Errorf("rate_limit.burst must be configured")

	case c.Timeout <= 0:
		return fmt.Errorf("timeout must be greater than 0")

	case c.Collection.LookbackDays < 0:
		return fmt.Errorf("lookback_days must be >= 0")
	}

	// API keys validation in a separate loop since it needs iteration
	for i, apiKey := range c.APIKeys {
		if apiKey.Key == "" {
			return fmt.Errorf("API key at position %d cannot be empty", i)
		}
	}

	return nil
}
