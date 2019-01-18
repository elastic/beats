// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pkg

// Config defines the package metricset's configuration options.
type Config struct {
}

// Validate validates the package metricset config.
func (c *Config) Validate() error {
	return nil
}

var defaultConfig = Config{}
