// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package cpu

import (
	"testing"

	"github.com/elastic/beats/v7/metricbeat/helper/prometheus/ptest"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	_ "github.com/elastic/beats/v7/x-pack/metricbeat/module/containerd"
)

func TestEventMapping(t *testing.T) {
	ptest.TestMetricSet(t, "containerd", "cpu",
		ptest.TestCases{
			{
				MetricsFile:  "../_meta/test/containerd.v1.5.2",
				ExpectedFile: "./_meta/test/containerd.v1.5.2.expected",
			},
		},
	)
}

func TestData(t *testing.T) {
	mbtest.TestDataFiles(t, "containerd", "cpu")
}
