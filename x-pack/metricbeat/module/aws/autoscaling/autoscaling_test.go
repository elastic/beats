// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package autoscaling

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConvertConfigMetricStats(t *testing.T) {
	cfg := []MetricConfigs{
		{[]string{"metric1"}, []string{"Maximum"}},
		{[]string{"metric2", "metric3"}, []string{"Average"}},
		{[]string{"metric4"}, []string{"Minimum"}},
	}
	result := convertConfigMetricStats(cfg)
	assert.Equal(t, result["metric1"], []string{"Maximum"})
	assert.Equal(t, result["metric2"], []string{"Average"})
	assert.Equal(t, result["metric3"], []string{"Average"})
	assert.Equal(t, result["metric4"], []string{"Minimum"})
}
