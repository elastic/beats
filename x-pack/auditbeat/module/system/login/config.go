// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package login

// Config defines the metricset's configuration options.
type Config struct {
	UtmpFilePattern string `config:"login.utmp_file_pattern" validate:"required"`
}

// Validate validates the login metricset config.
func (c *Config) Validate() error {
	return nil
}

var defaultConfig = Config{
	UtmpFilePattern: "/var/log/wtmp*",
}
