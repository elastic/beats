// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//
// The OTEL SDK stdoutmetric.Encoder used for console output is not concurrency
// safe. This wrapper controls access to Encoder to allow multiple endpoints
// to use same Exporter while prohibiting interweaving of exporter metrics in the
// output

package otel

import (
	"sync"

	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
)

// ConcurrentEncoder wraps a stdoutmetric.Encoder to serialize
// access to the underlying encoder.
// Allows the Exporter used for writing to console to be concurrent.
type ConcurrentEncoder struct {
	encoder stdoutmetric.Encoder
	lock    sync.Mutex
}

// NewConcurentEncoder creates a ConcurrentEncoder that wraps that
// stdoutmetric.Encoder that is passed in.
func NewConcurentEncoder(encoder stdoutmetric.Encoder) *ConcurrentEncoder {
	return &ConcurrentEncoder{
		encoder: encoder,
	}
}

// Encode enforces serial access to the underlying Encoder.
// Allows use by multiple processes without corrupting the state of the
// Encoder.
func (ce *ConcurrentEncoder) Encode(v any) error {
	ce.lock.Lock()
	defer ce.lock.Unlock()
	return ce.encoder.Encode(v)
}
