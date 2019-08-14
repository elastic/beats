// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitor

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFilterMetrics(t *testing.T) {
	selectedRange := []string{"TotalRequests", "Capacity", "CPUUsage"}
	intersection, difference := filterMetrics(selectedRange, MockMetricDefinitions())
	assert.Equal(t, intersection, []string{"TotalRequests", "Capacity"})
	assert.Equal(t, difference, []string{"CPUUsage"})
}
