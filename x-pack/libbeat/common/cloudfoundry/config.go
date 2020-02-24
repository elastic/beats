// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"time"

	"github.com/gofrs/uuid"
)

type Config struct {
	// CloudFoundry credentials for retrieving OAuth tokens
	ClientID     string `config:"client_id" validate:"required"`
	ClientSecret string `config:"client_secret" validate:"required"`

	// SkipVerify applies to all endpoints
	SkipVerify bool `config:"skip_verify"`

	// Override URLs returned from the CF client
	APIAddress     string `config:"api_address"`
	DopplerAddress string `config:"doppler_address"`
	UaaAddress     string `config:"uaa_address"`
	RlpAddress     string `config:"rlp_address"`

	// ShardID when retrieving events from loggregator, sharing this ID across
	// multiple filebeats will shard the load of receiving and sending events.
	ShardID string `config:"shard_id"`

	// Maximum amount of time to cache application objects from CF client
	CacheDuration time.Duration `config:"cache_duration"`
}

// InitDefaults initialize the defaults for the configuration.
func (c *Config) InitDefaults() {
	// If not provided by the user; subscription ID should be a unique string to avoid clustering by default.
	// Default to using a UUID4 string.
	uuid, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	c.ShardID = uuid.String()
	c.CacheDuration = 120 * time.Second
}
