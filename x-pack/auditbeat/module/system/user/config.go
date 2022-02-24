// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux && cgo
// +build linux,cgo

package user

import (
	"time"
)

// config defines the metricset's configuration options.
type config struct {
	StatePeriod           time.Duration `config:"state.period"`
	UserStatePeriod       time.Duration `config:"user.state.period"`
	DetectPasswordChanges bool          `config:"user.detect_password_changes"`
}

func (c *config) effectiveStatePeriod() time.Duration {
	if c.UserStatePeriod != 0 {
		return c.UserStatePeriod
	}
	return c.StatePeriod
}

func defaultConfig() config {
	return config{
		StatePeriod:           12 * time.Hour,
		DetectPasswordChanges: false,
	}
}
