// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package sqs

import (
	"fmt"
	"testing"

	"github.com/elastic/beats/x-pack/metricbeat/module/aws"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestFetch(t *testing.T) {
	config, info := aws.GetConfigForTest("sqs")
	if info != "" {
		t.Skip("Skipping TestFetch: " + info)
	}

	sqsMetricSet := mbtest.NewReportingMetricSetV2(t, config)
	events, err := mbtest.ReportingFetchV2(sqsMetricSet)
	if err != nil {
		t.Skip("Skipping TestFetch: failed to make api calls. Please check $AWS_ACCESS_KEY_ID, " +
			"$AWS_SECRET_ACCESS_KEY and $AWS_SESSION_TOKEN in config.yml")
	}

	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}
	t.Logf("Module: %s Metricset: %s", sqsMetricSet.Module().Name(), sqsMetricSet.Name())

	for _, event := range events {
		fmt.Println("event = ", event)
	}
}

func TestData(t *testing.T) {
	config, info := aws.GetConfigForTest("sqs")
	if info != "" {
		t.Skip("Skipping TestData: " + info)
	}

	sqsMetricSet := mbtest.NewReportingMetricSetV2(t, config)
	errs := mbtest.WriteEventsReporterV2(sqsMetricSet, t, "/")
	if errs != nil {
		t.Fatal("write", errs)
	}
}
