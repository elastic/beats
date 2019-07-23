// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleet

import (
	"time"

	xmanagement "github.com/elastic/beats/x-pack/libbeat/management"
)

// Config for central management
type Config struct {
	// true when enrolled
	Enabled bool `config:"enabled" yaml:"enabled"`

	Blacklist xmanagement.ConfigBlacklistSettings `config:"blacklist" yaml:"blacklist"`
}

// EventReporterConfig configuration of the events reporter.
type EventReporterConfig struct {
	Period       time.Duration `config:"period" yaml:"period"`
	MaxBatchSize int           `config:"max_batch_size" yaml:"max_batch_size" validate:"nonzero,positive"`
}

func defaultConfig() *Config {
	return &Config{
		Blacklist: xmanagement.ConfigBlacklistSettings{
			Patterns: map[string]string{
				"output": "console|file",
			},
		},
	}
}
