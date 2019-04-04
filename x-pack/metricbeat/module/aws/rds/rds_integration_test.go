// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package rds

import (
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws/mtest"
)

func TestFetch(t *testing.T) {
	config, info := mtest.GetConfigForTest("rds", "60s")
	if info != "" {
		t.Skip("Skipping TestFetch: " + info)
	}

	metricSet := mbtest.NewReportingMetricSetV2(t, config)
	events, errs := mbtest.ReportingFetchV2(metricSet)
	if errs != nil {
		t.Skip("Skipping TestFetch: failed to make api calls. Please check $AWS_ACCESS_KEY_ID, " +
			"$AWS_SECRET_ACCESS_KEY and $AWS_SESSION_TOKEN in config.yml")
	}

	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}
	t.Logf("Module: %s Metricset: %s", metricSet.Module().Name(), metricSet.Name())

	for _, event := range events {
		// RootField
		mtest.CheckEventField("service.name", "string", event, t)
		mtest.CheckEventField("cloud.provider", "string", event, t)
		mtest.CheckEventField("db_instance_arn", "string", event, t)
		mtest.CheckEventField("cloud.provider", "string", event, t)
		mtest.CheckEventField("cloud.region", "string", event, t)

		// MetricSetField
		mtest.CheckEventField("queries", "float", event, t)
		mtest.CheckEventField("latency.select", "float", event, t)
		mtest.CheckEventField("login_failures", "float", event, t)
	}
}

func TestData(t *testing.T) {
	config, info := mtest.GetConfigForTest("rds", "60s")
	if info != "" {
		t.Skip("Skipping TestData: " + info)
	}

	sqsMetricSet := mbtest.NewReportingMetricSetV2(t, config)
	errs := mbtest.WriteEventsReporterV2(sqsMetricSet, t, "/")
	if errs != nil {
		t.Fatal("write", errs)
	}
}
