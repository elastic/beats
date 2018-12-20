// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package host

import (
	"time"
)

// config defines the metricset's configuration options.
type config struct {
	StatePeriod     time.Duration `config:"state.period"`
	HostStatePeriod time.Duration `config:"host.state.period"`
}

func (c *config) effectiveStatePeriod() time.Duration {
	if c.HostStatePeriod != 0 {
		return c.HostStatePeriod
	}
	return c.StatePeriod
}

func defaultConfig() config {
	return config{
		StatePeriod: 1 * time.Hour,
	}
}
