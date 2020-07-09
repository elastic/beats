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

type ReloadConfig struct {
	Enabled bool          `config:"enabled" yaml:"enabled"`
	Period  time.Duration `config:"period" yaml:"period"`
}

func (r *ReloadConfig) Validate() error {
	if r.Enabled {
		if r.Period <= 0 {
			return ErrInvalidPeriod
		}
	}
	return nil
}
