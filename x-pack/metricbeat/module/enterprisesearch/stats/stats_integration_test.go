// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration
// +build integration

package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestFetch(t *testing.T) {
	service := compose.EnsureUpWithTimeout(t, 300, "enterprise_search")

	config := getConfig("stats", service.Host())
	f := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 errors, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
	event := events[0].MetricSetFields
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)
}

func TestData(t *testing.T) {
	service := compose.EnsureUpWithTimeout(t, 300, "enterprise_search")

	config := getConfig("stats", service.Host())

	f := mbtest.NewReportingMetricSetV2Error(t, config)
	err := mbtest.WriteEventsReporterV2Error(f, t, "")
	if err != nil {
		t.Fatal("write", err)
	}
}

// GetConfig returns config for Enterprise Search module
func getConfig(metricset string, host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "enterprisesearch",
		"metricsets": []string{metricset},
		"hosts":      []string{host},
		"username":   "elastic",
		"password":   "changeme",
	}
}
