// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package s3_request

import (
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws/mtest"
)

func TestFetch(t *testing.T) {
	config, info := mtest.GetConfigForTest("s3_request", "3600s")
	if info != "" {
		t.Skip("Skipping TestFetch: " + info)
	}

	s3DailyMetricSet := mbtest.NewReportingMetricSetV2(t, config)
	events, err := mbtest.ReportingFetchV2(s3DailyMetricSet)
	if err != nil {
		t.Skip("Skipping TestFetch: failed to make api calls. Please check $AWS_ACCESS_KEY_ID, " +
			"$AWS_SECRET_ACCESS_KEY and $AWS_SESSION_TOKEN in config.yml")
	}

	assert.Empty(t, err)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}
	t.Logf("Module: %s Metricset: %s", s3DailyMetricSet.Module().Name(), s3DailyMetricSet.Name())

	for _, event := range events {
		// RootField
		mtest.CheckEventField("service.name", "string", event, t)
		mtest.CheckEventField("cloud.region", "string", event, t)

		// MetricSetField
		mtest.CheckEventField("bucket.name", "string", event, t)
		mtest.CheckEventField("all_requests", "int", event, t)
		mtest.CheckEventField("get_requests", "int", event, t)
		mtest.CheckEventField("put_requests", "int", event, t)
		mtest.CheckEventField("delete_requests", "int", event, t)
		mtest.CheckEventField("head_requests", "int", event, t)
		mtest.CheckEventField("post_requests", "int", event, t)
		mtest.CheckEventField("select_requests", "int", event, t)
		mtest.CheckEventField("select_scanned.bytes", "float", event, t)
		mtest.CheckEventField("select_returned.bytes", "float", event, t)
		mtest.CheckEventField("list_requests", "int", event, t)
		mtest.CheckEventField("bytes_downloaded", "float", event, t)
		mtest.CheckEventField("bytes_uploaded", "float", event, t)
		mtest.CheckEventField("4xx_errors", "int", event, t)
		mtest.CheckEventField("5xx_errors", "int", event, t)
		mtest.CheckEventField("first_byte_latency", "float", event, t)
		mtest.CheckEventField("total_request_latency", "float", event, t)
	}
}

func TestData(t *testing.T) {
	config, info := mtest.GetConfigForTest("s3_request", "300s")
	if info != "" {
		t.Skip("Skipping TestData: " + info)
	}

	ec2MetricSet := mbtest.NewReportingMetricSetV2(t, config)
	errs := mbtest.WriteEventsReporterV2(ec2MetricSet, t, "/")
	if errs != nil {
		t.Fatal("write", errs)
	}
}
