// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package docker

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/common/docker"
)

// Config for docker provider
type Config struct {
	Host           string            `config:"host"`
	TLS            *docker.TLSConfig `config:"ssl"`
	CleanupTimeout time.Duration     `config:"cleanup_timeout" validate:"positive"`
}

// InitDefaults initializes the default values for the config.
func (c *Config) InitDefaults() {
	c.Host = "unix:///var/run/docker.sock"
	c.CleanupTimeout = 60 * time.Second
}
