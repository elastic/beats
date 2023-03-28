// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/monitoring"
)

// TestInputMetricsClose asserts that metrics registered by this input are
// removed after Close() is called. This is important because an input with
// the same ID could be re-registered, and that ID cannot exist in the
// monitoring registry.
func TestInputMetricsClose(t *testing.T) {
	reg := monitoring.NewRegistry()

	metrics := newInputMetrics("aws-s3-aws.cloudfront_logs-8b312b5f-9f99-492c-b035-3dff354a1f01", reg, 1)
	metrics.Close()

	reg.Do(monitoring.Full, func(s string, _ interface{}) {
		t.Errorf("registry should be empty, but found %v", s)
	})
}

func TestInputMetricsSQSWorkerUtilization(t *testing.T) {
	reg := monitoring.NewRegistry()
	maxWorkers := 1
	metrics := newInputMetrics("test", reg, maxWorkers)

	metrics.utilizationNanos = time.Second.Nanoseconds()

	utilization := calculateUtilization(time.Second, maxWorkers, metrics)
	assert.Greater(t, utilization, 0.95)
}
