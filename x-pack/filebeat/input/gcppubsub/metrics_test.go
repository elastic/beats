// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcppubsub

import (
	"testing"

	"github.com/elastic/elastic-agent-libs/monitoring"
)

// TestInputMetricsClose asserts that metrics registered by this input are
// removed after Close() is called. This is important because an input with
// the same ID could be re-registered, and that ID cannot exist in the
// monitoring registry.
func TestInputMetricsClose(t *testing.T) {
	reg := monitoring.NewRegistry()

	metrics := newInputMetrics("gcp", reg)
	metrics.Close()

	reg.Do(monitoring.Full, func(s string, _ interface{}) {
		t.Errorf("registry should be empty, but found %v", s)
	})
}
