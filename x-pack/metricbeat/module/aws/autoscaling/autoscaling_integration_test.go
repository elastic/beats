// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && aws
// +build integration,aws

package autoscaling

import (
	"testing"

	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws/mtest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/assert"
)

const metricSetName = "autoscaling"

func TestFetchAllMetrics(t *testing.T) {
	config := mtest.GetConfigForTest(t, metricSetName, "5m")
	getEvents(t, config)
}

func TestFetchSpecificMetric(t *testing.T) {
	metricName := "GroupDesiredCapacity"
	stat := "Minimum"
	config := mtest.GetConfigForTest(t, metricSetName, "5m")
	config = addAutoscalingMetricsToConfig(config, metricName, stat)
	events := getEvents(t, config)
	for _, e := range events {
		metrics, err := e.RootFields.GetValue("aws.autoscaling.metrics")
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		metricNames := metrics.(mapstr.M).FlattenKeys()
		// there must be only one metric name
		assert.Equal(t, *metricNames, []string{metricName + ".min", metricName})
	}
}

func TestData(t *testing.T) {
	config := mtest.GetConfigForTest(t, metricSetName, "5m")
	metricSet := mbtest.NewFetcher(t, config)
	metricSet.WriteEvents(t, "/")
}

func getEvents(t *testing.T, config map[string]interface{}) []mb.Event {
	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(metricSet)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
	mbtest.TestMetricsetFieldsDocumented(t, metricSet, events)
	return events
}

func addAutoscalingMetricsToConfig(config map[string]interface{}, metricName string, stat string) map[string]interface{} {
	metricsConfig := []map[string]interface{}{}
	cloudwatchMetric := map[string]interface{}{}
	cloudwatchMetric["name"] = metricName
	cloudwatchMetric["statistic"] = stat
	metricsConfig = append(metricsConfig, cloudwatchMetric)
	config["metrics"] = metricsConfig
	return config
}
