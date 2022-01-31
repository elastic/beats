// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package retry

import (
	"context"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/backoff"
)

// DoWithBackoff ignores retry config of delays and lets backoff decide how much time it needs.
func DoWithBackoff(config *Config, b backoff.Backoff, fn func() error, fatalErrors ...error) error {
	retryCount := getRetryCount(config)
	var err error

	for retryNo := 0; retryNo <= retryCount; retryNo++ {
		err = fn()
		if err == nil || isFatal(err, fatalErrors...) {
			b.Reset()
			return err
		}

		if retryNo < retryCount {
			b.Wait()
		}
	}

	return err
}

// Do runs provided function in a manner specified in retry configuration
func Do(ctx context.Context, config *Config, fn func(ctx context.Context) error, fatalErrors ...error) error {
	retryCount := getRetryCount(config)
	var err error

RETRY_LOOP:
	for retryNo := 0; retryNo <= retryCount; retryNo++ {
		if ctx.Err() != nil {
			break
		}

		err = fn(ctx)
		if err == nil {
			return nil
		}

		if isFatal(err, fatalErrors...) {
			return err
		}

		if retryNo < retryCount {
			t := time.NewTimer(getDelayDuration(config, retryNo))
			select {
			case <-t.C:
			case <-ctx.Done():
				t.Stop()
				break RETRY_LOOP
			}
		}
	}

	return err
}

func getRetryCount(config *Config) int {
	if config == nil {
		return defaultRetriesCount
	}

	if !config.Enabled {
		return 0
	}

	if config.RetriesCount > 0 {
		return config.RetriesCount
	}

	return defaultRetriesCount
}

func getDelayDuration(config *Config, retryNo int) time.Duration {
	delay := defaultDelay

	if config != nil {
		if config.Delay > 0 {
			delay = config.Delay
		}

		if config.Exponential {
			delay = time.Duration(delay.Nanoseconds() * int64(retryNo+1))
		}
	}

	maxDelay := config.MaxDelay
	if maxDelay == 0 {
		maxDelay = defaultMaxDelay
	}
	if delay > maxDelay {
		delay = maxDelay
	}
	return time.Duration(delay)
}

// Error is fatal either if it implements Error interface and says so
// or if it is equal to one of the fatal values provided
func isFatal(err error, fatalErrors ...error) bool {
	if fatalerr, ok := err.(Fatal); ok {
		return fatalerr.Fatal()
	}

	for _, e := range fatalErrors {
		if e == err {
			return true
		}
	}

	// What does not match criteria is considered transient
	return false
}
