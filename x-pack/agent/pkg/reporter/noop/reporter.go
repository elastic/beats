// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package noop

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/reporter"
)

// Reporter is a reporter without any effects, serves just as a showcase for further implementations.
type Reporter struct{}

// NewReporter creates a new noop reporter
func NewReporter() *Reporter {
	return &Reporter{}
}

// Report in noop reporter does nothing
func (*Reporter) Report(_ context.Context, _ reporter.Event) error { return nil }

// Close stops all the background jobs reporter is running.
func (*Reporter) Close() error { return nil }

// Check it is reporter.Backend
var _ reporter.Backend = &Reporter{}
