// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
)

const (
	ConsumerVersionV1 = "v1"
	ConsumerVersionV2 = "v2"
)

type Config struct {
	// Version of the consumer to use, it can be v1 or v2, defaults to v1
	Version string `config:"version"`

	// CloudFoundry credentials for retrieving OAuth tokens
	ClientID     string `config:"client_id" validate:"required"`
	ClientSecret string `config:"client_secret" validate:"required"`

	// TLS configuration for the client
	TLS *tlscommon.Config `config:"ssl"`

	// Override URLs returned from the CF client
	APIAddress     string `config:"api_address" validate:"required"`
	DopplerAddress string `config:"doppler_address"`
	UaaAddress     string `config:"uaa_address"`
	RlpAddress     string `config:"rlp_address"`

	// ShardID when retrieving events from loggregator, sharing this ID across
	// multiple filebeats will shard the load of receiving and sending events.
	ShardID string `config:"shard_id" validate:"required"`

	// Maximum amount of time to cache application objects from CF client.
	CacheDuration time.Duration `config:"cache_duration"`

	// Time to wait before retrying to get application info in case of error.
	CacheRetryDelay time.Duration `config:"cache_retry_delay"`
}

// InitDefaults initialize the defaults for the configuration.
func (c *Config) InitDefaults() {
	c.CacheDuration = 120 * time.Second
	c.CacheRetryDelay = 20 * time.Second
	c.Version = ConsumerVersionV1
}

func (c *Config) Validate() error {
	supportedVersions := []string{ConsumerVersionV1, ConsumerVersionV2}
	if !anyOf(supportedVersions, c.Version) {
		return fmt.Errorf("not supported version %v, expected one of %s", c.Version, strings.Join(supportedVersions, ", "))
	}
	return nil
}

// TLSConfig returns the TLS configuration.
func (c *Config) TLSConfig() (*tls.Config, error) {
	tls, err := tlscommon.LoadTLSConfig(c.TLS)
	if err != nil {
		return nil, err
	}
	return tls.ToConfig(), nil
}

func anyOf(elems []string, s string) bool {
	for _, elem := range elems {
		if s == elem {
			return true
		}
	}
	return false
}
