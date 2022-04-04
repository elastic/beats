// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package collector

import (
	"testing"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"

	_ "github.com/elastic/beats/v7/x-pack/metricbeat/module/prometheus"

	// Import common fields for validation
	_ "github.com/elastic/beats/v7/metricbeat/module/prometheus"
)

func TestData(t *testing.T) {
	mbtest.TestDataFiles(t, "prometheus", "collector")
}
