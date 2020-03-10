// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration
// +build aws

package rds

import (
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws/mtest"
)

func TestFetch(t *testing.T) {
	config := mtest.GetConfigForTest(t, "rds", "60s")

	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(metricSet)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}

	assert.NotEmpty(t, events)

	for _, event := range events {
		t.Logf("%s/%s event: %+v", metricSet.Module().Name(), metricSet.Name(), event)

		// RootField
		mtest.CheckEventField("service.name", "string", event, t)
		mtest.CheckEventField("cloud.provider", "string", event, t)
		mtest.CheckEventField("cloud.provider", "string", event, t)
		mtest.CheckEventField("cloud.region", "string", event, t)
		mtest.CheckEventField("cloud.availability_zone", "string", event, t)

		// MetricSetField
		mtest.CheckEventField("db_instance.arn", "string", event, t)
		mtest.CheckEventField("db_instance.class", "string", event, t)
		mtest.CheckEventField("queries", "float", event, t)
		mtest.CheckEventField("latency.select", "float", event, t)
		mtest.CheckEventField("login_failures", "float", event, t)
	}
}

func TestData(t *testing.T) {
	config := mtest.GetConfigForTest(t, "rds", "60s")

	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	if err := mbtest.WriteEventsReporterV2Error(metricSet, t, "/"); err != nil {
		t.Fatal("write", err)
	}
}
