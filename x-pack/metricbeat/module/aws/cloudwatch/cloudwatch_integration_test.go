// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && aws
// +build integration,aws

package cloudwatch

import (
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"
	"github.com/elastic/beats/v8/x-pack/metricbeat/module/aws/mtest"
)

func TestFetch(t *testing.T) {
	config := mtest.GetConfigForTest(t, "cloudwatch", "300s")

	config = addCloudwatchMetricsToConfig(config)
	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(metricSet)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}

	assert.NotEmpty(t, events)
	mbtest.TestMetricsetFieldsDocumented(t, metricSet, events)
}

func TestData(t *testing.T) {
	config := mtest.GetConfigForTest(t, "cloudwatch", "300s")

	config = addCloudwatchMetricsToConfig(config)
	metricSet := mbtest.NewFetcher(t, config)
	metricSet.WriteEvents(t, "/")
}

func addCloudwatchMetricsToConfig(config map[string]interface{}) map[string]interface{} {
	cloudwatchMetricsConfig := []map[string]interface{}{}
	cloudwatchMetric := map[string]interface{}{}
	cloudwatchMetric["namespace"] = "AWS/RDS"
	cloudwatchMetricsConfig = append(cloudwatchMetricsConfig, cloudwatchMetric)
	config["metrics"] = cloudwatchMetricsConfig
	return config
}
