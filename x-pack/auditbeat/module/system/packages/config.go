// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package packages

// Config defines the host metricset's configuration options.
type Config struct {
	ReportChanges bool `config:"packages.report_changes"`
}

// Validate validates the host metricset config.
func (c *Config) Validate() error {
	return nil
}

var defaultConfig = Config{
	ReportChanges: true,
}
