// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration
// +build aws

package sqs

import (
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws/mtest"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestFetch(t *testing.T) {
	config := mtest.GetConfigForTest(t, "sqs", "300s")

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
		mtest.CheckEventField("empty_receives", "float", event, t)
		mtest.CheckEventField("messages.delayed", "float", event, t)
		mtest.CheckEventField("messages.deleted", "float", event, t)
		mtest.CheckEventField("messages.not_visible", "float", event, t)
		mtest.CheckEventField("messages.received", "float", event, t)
		mtest.CheckEventField("messages.sent", "float", event, t)
		mtest.CheckEventField("messages.visible", "float", event, t)
		mtest.CheckEventField("oldest_message_age.sec", "float", event, t)
		mtest.CheckEventField("sent_message_size", "float", event, t)
		mtest.CheckEventField("queue.name", "string", event, t)
	}
}

func TestData(t *testing.T) {
	config := mtest.GetConfigForTest(t, "sqs", "300s")

	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	if err := mbtest.WriteEventsReporterV2Error(metricSet, t, "/"); err != nil {
		t.Fatal("write", err)
	}
}
