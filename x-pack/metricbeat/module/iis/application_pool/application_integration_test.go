// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration
// +build windows

package application_pool

import (
	"testing"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestFetch(t *testing.T) {
	config := map[string]interface{}{
		"module":     "iis",
		"period":     "30s",
		"metricsets": []string{"application_pool"},
	}
	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	_, errs := mbtest.ReportingFetchV2Error(metricSet)
	if len(errs) > 0 {
		// should find a way to first check if iis is running
		//t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
}

func TestData(t *testing.T) {
	config := map[string]interface{}{
		"module":     "iis",
		"period":     "30s",
		"metricsets": []string{"application_pool"},
	}

	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	if err := mbtest.WriteEventsReporterV2Error(metricSet, t, "/"); err != nil {
		// should find a way to first check if iis is running
		//	t.Fatal("write", err)
	}
}
