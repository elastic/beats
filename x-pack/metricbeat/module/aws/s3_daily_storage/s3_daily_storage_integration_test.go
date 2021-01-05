// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration
// +build aws

package s3_daily_storage

import (
	"testing"

	"github.com/stretchr/testify/assert"

	_ "github.com/elastic/beats/v7/libbeat/processors/actions"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws/mtest"
)

func TestFetch(t *testing.T) {
	config := mtest.GetConfigForTest(t, "s3_daily_storage", "86400s")

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
		mtest.CheckEventField("aws.s3.metrics.BucketSizeBytes.avg", "float", event, t)
		mtest.CheckEventField("aws.s3.metrics.NumberOfObjects.avg", "float", event, t)
	}
}

func TestData(t *testing.T) {
	config := mtest.GetConfigForTest(t, "s3_daily_storage", "86400s")

	metricSet := mbtest.NewFetcher(t, config)
	metricSet.WriteEvents(t, "/")
}
