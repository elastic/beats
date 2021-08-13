// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanMetricNameString(t *testing.T) {
	computeMC := metricsConfig{"compute", "", []string{}, ""}

	cases := []struct {
		title              string
		metricType         string
		aligner            string
		expectedMetricName string
	}{
		{
			"test construct metric name with ALIGN_MEAN aligner",
			"compute.googleapis.com/instance/cpu/usage_time",
			"ALIGN_MEAN",
			"instance.cpu.usage_time.avg",
		},
		{
			"test construct metric name with ALIGN_NONE aligner",
			"compute.googleapis.com/instance/cpu/utilization",
			"ALIGN_NONE",
			"instance.cpu.utilization.value",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			metricName := cleanMetricNameString(c.metricType, c.aligner, computeMC)
			assert.Equal(t, c.expectedMetricName, metricName)
		})
	}
}
