// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package socket

import (
	"time"
)

// Config defines the socket metricset's configuration options.
type Config struct {
	StatePeriod       time.Duration `config:"state.period"`
	SocketStatePeriod time.Duration `config:"socket.state.period"`
	IncludeLocalhost  bool          `config:"socket.include_localhost"`
}

// Validate validates the host metricset config.
func (c *Config) Validate() error {
	return nil
}

func (c *Config) effectiveStatePeriod() time.Duration {
	if c.SocketStatePeriod != 0 {
		return c.SocketStatePeriod
	}
	return c.StatePeriod
}

var defaultConfig = Config{
	StatePeriod:      1 * time.Hour,
	IncludeLocalhost: false,
}
