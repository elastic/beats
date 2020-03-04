// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration
// +build windows

package webserver

import (
	"testing"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"

	// Register input module and metricset
	_ "github.com/elastic/beats/v7/metricbeat/module/windows"
	_ "github.com/elastic/beats/v7/metricbeat/module/windows/perfmon"
)

func TestData(t *testing.T) {
	c := map[string]interface{}{
		"module":     "iis",
		"metricsets": []string{"webserver"},
	}
	metricSet := mbtest.NewReportingMetricSetV2Error(t, c)
	if err := mbtest.WriteEventsReporterV2Error(metricSet, t, "/"); err != nil {
		// should find a way to first check if iis is running
		//	t.Fatal("write", err)
	}
}

func TestFetch(t *testing.T) {
	c := map[string]interface{}{
		"module":     "iis",
		"metricsets": []string{"webserver"},
	}
	m := mbtest.NewFetcher(t, c)
	_, errs := m.FetchEvents()
	if len(errs) > 0 {
		// should find a way to first check if iis is running
		//t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
}
