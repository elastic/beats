// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && gcp
// +build integration,gcp

package metrics

import (
	"testing"

	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	config := GetConfigForTest(t, "metrics")
	metricSet := mbtest.NewFetcher(t, config)
	metricSet.WriteEvents(t, "/")
}
