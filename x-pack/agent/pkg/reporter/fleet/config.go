// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleet

// ManagementConfig is a configuration describing fleet connected parts
type ManagementConfig struct {
	Enabled                 bool `config:"enabled"`
	Threshold               int  `config:"threshold" validate:"min=1"`
	ReportingCheckFrequency int  `config:"check_frequency_sec" validate:"min=1"`
}

// DefaultFleetManagementConfig initiates FleetManagementConfig with default values
func DefaultFleetManagementConfig() *ManagementConfig {
	return &ManagementConfig{
		Enabled:                 false,
		Threshold:               10000,
		ReportingCheckFrequency: 30,
	}
}
