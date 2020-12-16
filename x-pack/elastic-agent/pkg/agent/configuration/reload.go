// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package configuration

import (
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
)

var (
	// ErrInvalidPeriod is returned when a reload period interval is not valid
	ErrInvalidPeriod = errors.New("period must be higher than zero")
)

// ReloadConfig defines behavior of a reloader for standalone configuration.
type ReloadConfig struct {
	Enabled bool          `config:"enabled" yaml:"enabled"`
	Period  time.Duration `config:"period" yaml:"period"`
}

// Validate validates settings of configuration.
func (r *ReloadConfig) Validate() error {
	if r.Enabled {
		if r.Period <= 0 {
			return ErrInvalidPeriod
		}
	}
	return nil
}

// DefaultReloadConfig creates a default configuration for standalone mode.
func DefaultReloadConfig() *ReloadConfig {
	return &ReloadConfig{
		Enabled: true,
		Period:  10 * time.Second,
	}
}
