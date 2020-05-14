// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration
// +build aws

package s3_request

import (
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws/mtest"
)

func TestFetch(t *testing.T) {
	config := mtest.GetConfigForTest(t, "s3_request", "86400s")

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
		mtest.CheckEventField("requests.total", "int", event, t)
		mtest.CheckEventField("requests.get", "int", event, t)
		mtest.CheckEventField("requests.put", "int", event, t)
		mtest.CheckEventField("requests.delete", "int", event, t)
		mtest.CheckEventField("requests.head", "int", event, t)
		mtest.CheckEventField("requests.post", "int", event, t)
		mtest.CheckEventField("select.requests", "int", event, t)
		mtest.CheckEventField("select_scanned.bytes", "float", event, t)
		mtest.CheckEventField("select_returned.bytes", "float", event, t)
		mtest.CheckEventField("requests.list", "int", event, t)
		mtest.CheckEventField("downloaded.bytes", "float", event, t)
		mtest.CheckEventField("uploaded.bytes", "float", event, t)
		mtest.CheckEventField("errors.4xx", "int", event, t)
		mtest.CheckEventField("errors.5xx", "int", event, t)
		mtest.CheckEventField("latency.first_byte.ms", "float", event, t)
		mtest.CheckEventField("latency.total_request.ms", "float", event, t)
	}
}

func TestData(t *testing.T) {
	config := mtest.GetConfigForTest(t, "s3_request", "86400s")

	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	if err := mbtest.WriteEventsReporterV2Error(metricSet, t, "/"); err != nil {
		t.Fatal("write", err)
	}
}
