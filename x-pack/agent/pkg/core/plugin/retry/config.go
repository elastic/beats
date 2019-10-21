// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package retry

import "time"

const (
	defaultRetriesCount = 3
	defaultDelay        = 30 * time.Second
	defaultMaxDelay     = 5 * time.Minute
)

// Config is a configuration of a strategy
type Config struct {
	// Enabled determines whether retry is possible. Default is false.
	Enabled bool `yaml:"enabled" config:"enabled"`
	// RetriesCount specifies number of retries. Default is 3.
	// Retry count of 1 means it will be retried one time after one failure.
	RetriesCount int `yaml:"retriesCount" config:"retriesCount"`
	// Delay specifies delay in ms between retries. Default is 30s
	Delay time.Duration `yaml:"delay" config:"delay"`
	// MaxDelay specifies maximum delay in ms between retries. Default is 300s
	MaxDelay time.Duration `yaml:"maxDelay" config:"maxDelay"`
	// Exponential determines whether delay is treated as exponential.
	// With 30s delay and 3 retries: 30, 60, 120s
	// Default is false
	Exponential bool `yaml:"exponential" config:"exponential"`
}
