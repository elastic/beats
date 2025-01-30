// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && aws

package sqs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	_ "github.com/elastic/beats/v7/libbeat/processors/actions"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws/mtest"
)

func TestFetch(t *testing.T) {
	config := mtest.GetConfigForTest(t, "sqs", "300s")

	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.PeriodicReportingFetchV2Error(metricSet, 1*time.Minute, 8*time.Minute)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}

	assert.NotEmpty(t, events)
	mbtest.TestMetricsetFieldsDocumented(t, metricSet, events)
}

func TestData(t *testing.T) {
	config := mtest.GetConfigForTest(t, "sqs", "300s")

	metricSet := mbtest.NewFetcher(t, config)
	metricSet.WriteEvents(t, "/")
}
