// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package grpc

import (
	"github.com/elastic/beats/libbeat/common/backoff"
)

// NoopBackoff implements a backoff interface without any wait.
// Used when no backoff is configured.
type NoopBackoff struct{}

// NewEqualJitterBackoff returns a new EqualJitter object.
func NewNoopBackoff() backoff.Backoff {
	return &NoopBackoff{}
}

// Reset resets the duration of the backoff.
func (b *NoopBackoff) Reset() {}

// Wait block until either the timer is completed or channel is done.
func (b *NoopBackoff) Wait() bool {
	return true
}
