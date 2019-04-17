// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package cloudwatch

import (
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws/mtest"
)

func TestFetch(t *testing.T) {
	config, info := mtest.GetConfigForTest("cloudwatch", "300s")
	if info != "" {
		t.Skip("Skipping TestFetch: " + info)
	}

	config = addCloudwatchMetricsToConfig(config)
	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(metricSet)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}

	assert.NotEmpty(t, events)
}

func TestData(t *testing.T) {
	config, info := mtest.GetConfigForTest("cloudwatch", "300s")
	if info != "" {
		t.Skip("Skipping TestData: " + info)
	}

	config = addCloudwatchMetricsToConfig(config)
	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	if err := mbtest.WriteEventsReporterV2Error(metricSet, t, "/"); err != nil {
		t.Fatal("write", err)
	}
}

func addCloudwatchMetricsToConfig(config map[string]interface{}) map[string]interface{} {
	cloudwatchMetricsConfig := []map[string]interface{}{}
	cloudwatchMetric := map[string]interface{}{}
	cloudwatchMetric["namespace"] = "AWS/RDS"
	cloudwatchMetricsConfig = append(cloudwatchMetricsConfig, cloudwatchMetric)
	config["cloudwatch_metrics"] = cloudwatchMetricsConfig
	return config
}
