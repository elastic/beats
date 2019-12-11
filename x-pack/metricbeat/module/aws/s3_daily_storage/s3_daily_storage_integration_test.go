// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package s3_daily_storage

import (
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws/mtest"
)

func TestFetch(t *testing.T) {
	config, info := mtest.GetConfigForTest("s3_daily_storage", "86400s")
	if info != "" {
		t.Skip("Skipping TestFetch: " + info)
	}

	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(metricSet)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}

	assert.NotEmpty(t, events)

	for _, event := range events {
		// RootField
		mtest.CheckEventField("service.name", "string", event, t)
		mtest.CheckEventField("cloud.region", "string", event, t)

		// MetricSetField
		mtest.CheckEventField("bucket.name", "string", event, t)
		mtest.CheckEventField("bucket.size.bytes", "float", event, t)
		mtest.CheckEventField("number_of_objects", "float", event, t)
	}
}

func TestData(t *testing.T) {
	config, info := mtest.GetConfigForTest("s3_daily_storage", "86400s")
	if info != "" {
		t.Skip("Skipping TestData: " + info)
	}

	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	if err := mbtest.WriteEventsReporterV2Error(metricSet, t, "/"); err != nil {
		t.Fatal("write", err)
	}
}
