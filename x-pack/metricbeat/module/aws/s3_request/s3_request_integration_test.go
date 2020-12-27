// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration
// +build aws

package s3_request

import (
	"testing"

	"github.com/stretchr/testify/assert"

	_ "github.com/elastic/beats/v7/libbeat/processors/actions"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws/mtest"
)

func TestFetch(t *testing.T) {
	t.Skip("flaky test: https://github.com/elastic/beats/issues/21826")
	config := mtest.GetConfigForTest(t, "s3_request", "60s")

	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(metricSet)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}

	assert.NotEmpty(t, events)

	for _, event := range events {
		mtest.CheckEventField("cloud.region", "string", event, t)
		mtest.CheckEventField("aws.dimensions.BucketName", "string", event, t)
		mtest.CheckEventField("aws.dimensions.StorageType", "string", event, t)
		mtest.CheckEventField("s3.metrics.AllRequests.avg", "int", event, t)
		mtest.CheckEventField("s3.metrics.GetRequests.avg", "int", event, t)
		mtest.CheckEventField("s3.metrics.PutRequests.avg", "int", event, t)
		mtest.CheckEventField("s3.metrics.DeleteRequests.avg", "int", event, t)
		mtest.CheckEventField("s3.metrics.HeadRequests.avg", "int", event, t)
		mtest.CheckEventField("s3.metrics.PostRequests.avg", "int", event, t)
		mtest.CheckEventField("s3.metrics.SelectRequests.avg", "int", event, t)
		mtest.CheckEventField("s3.metrics.SelectScannedBytes.avg", "float", event, t)
		mtest.CheckEventField("s3.metrics.SelectReturnedBytes.avg", "float", event, t)
		mtest.CheckEventField("s3.metrics.ListRequests.avg", "int", event, t)
		mtest.CheckEventField("s3.metrics.BytesDownloaded.avg", "float", event, t)
		mtest.CheckEventField("s3.metrics.BytesUploaded.avg", "float", event, t)
		mtest.CheckEventField("s3.metrics.4xxErrors.avg", "int", event, t)
		mtest.CheckEventField("s3.metrics.5xxErrors.avg", "int", event, t)
		mtest.CheckEventField("s3.metrics.FirstByteLatency.avg", "float", event, t)
		mtest.CheckEventField("s3.metrics.TotalRequestLatency.avg", "float", event, t)
	}
}

func TestData(t *testing.T) {
	config := mtest.GetConfigForTest(t, "s3_request", "60s")

	metricSet := mbtest.NewFetcher(t, config)
	metricSet.WriteEvents(t, "/")
}
