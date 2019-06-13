// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package process

import (
	"time"
)

// Config defines the host metricset's configuration options.
type Config struct {
	StatePeriod        time.Duration `config:"state.period"`
	ProcessStatePeriod time.Duration `config:"process.state.period"`
}

// Validate validates the host metricset config.
func (c *Config) Validate() error {
	return nil
}

func (c *Config) effectiveStatePeriod() time.Duration {
	if c.ProcessStatePeriod != 0 {
		return c.ProcessStatePeriod
	}
	return c.StatePeriod
}

var defaultConfig = Config{
	StatePeriod: 12 * time.Hour,
}
