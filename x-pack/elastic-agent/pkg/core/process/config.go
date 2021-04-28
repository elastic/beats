// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package process

import "time"

// Config for fine tuning new process
type Config struct {
	SpawnTimeout   time.Duration `yaml:"spawn_timeout" config:"spawn_timeout"`
	StopTimeout    time.Duration `yaml:"stop_timeout" config:"stop_timeout"`
	FailureTimeout time.Duration `yaml:"failure_timeout" config:"failure_timeout"`

	// TODO: cgroups and namespaces
}

// DefaultConfig creates a config with pre-set default values.
func DefaultConfig() *Config {
	return &Config{
		SpawnTimeout:   30 * time.Second,
		StopTimeout:    30 * time.Second,
		FailureTimeout: 10 * time.Second,
	}
}
