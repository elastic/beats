// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/cloudwatchiface"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	"github.com/elastic/beats/metricbeat/mb"
)

// GetStartTimeEndTime function uses durationString to create startTime and endTime for queries.
func GetStartTimeEndTime(durationString string) (startTime time.Time, endTime time.Time, err error) {
	endTime = time.Now()
	duration, err := time.ParseDuration(durationString)
	if err != nil {
		return
	}
	startTime = endTime.Add(duration)
	return startTime, endTime, nil
}

// GetListMetricsOutput function gets listMetrics results from cloudwatch per namespace for each region.
// ListMetrics Cloudwatch API is used to list the specified metrics. The returned metrics can be used with GetMetricData
// to obtain statistical data.
func GetListMetricsOutput(namespace string, regionName string, svcCloudwatch cloudwatchiface.CloudWatchAPI) ([]cloudwatch.Metric, error) {
	listMetricsInput := &cloudwatch.ListMetricsInput{Namespace: &namespace}
	reqListMetrics := svcCloudwatch.ListMetricsRequest(listMetricsInput)

	// List metrics of a given namespace for each region
	listMetricsOutput, err := reqListMetrics.Send()
	if err != nil {
		err = errors.Wrap(err, "ListMetricsRequest failed, skipping region "+regionName)
		return nil, err
	}

	if listMetricsOutput.Metrics == nil || len(listMetricsOutput.Metrics) == 0 {
		// No metrics in this region
		return nil, nil
	}
	return listMetricsOutput.Metrics, nil
}

func getMetricDataPerRegion(metricDataQueries []cloudwatch.MetricDataQuery, nextToken *string, svc cloudwatchiface.CloudWatchAPI, startTime time.Time, endTime time.Time) (*cloudwatch.GetMetricDataOutput, error) {
	getMetricDataInput := &cloudwatch.GetMetricDataInput{
		NextToken:         nextToken,
		StartTime:         &startTime,
		EndTime:           &endTime,
		MetricDataQueries: metricDataQueries,
	}

	reqGetMetricData := svc.GetMetricDataRequest(getMetricDataInput)
	getMetricDataOutput, err := reqGetMetricData.Send()
	if err != nil {
		err = errors.Wrap(err, "Error GetMetricDataInput")
		return nil, err
	}
	return getMetricDataOutput, nil
}

// GetMetricDataResults function uses MetricDataQueries to get metric data output.
func GetMetricDataResults(metricDataQueries []cloudwatch.MetricDataQuery, svc cloudwatchiface.CloudWatchAPI, startTime time.Time, endTime time.Time) ([]cloudwatch.MetricDataResult, error) {
	init := true
	getMetricDataOutput := &cloudwatch.GetMetricDataOutput{NextToken: nil}
	for init || getMetricDataOutput.NextToken != nil {
		init = false
		output, err := getMetricDataPerRegion(metricDataQueries, getMetricDataOutput.NextToken, svc, startTime, endTime)
		if err != nil {
			err = errors.Wrap(err, "getMetricDataPerRegion failed")
			return getMetricDataOutput.MetricDataResults, err
		}
		getMetricDataOutput.MetricDataResults = append(getMetricDataOutput.MetricDataResults, output.MetricDataResults...)
	}
	return getMetricDataOutput.MetricDataResults, nil
}

// EventMapping maps data in input to a predefined schema.
func EventMapping(input map[string]interface{}, schema s.Schema) (common.MapStr, error) {
	return schema.Apply(input, s.FailOnRequired)
}

// GetConfigForTest function gets aws credentials for integration tests.
func GetConfigForTest(metricSetName string) (map[string]interface{}, string) {
	accessKeyID, okAccessKeyID := os.LookupEnv("AWS_ACCESS_KEY_ID")
	secretAccessKey, okSecretAccessKey := os.LookupEnv("AWS_SECRET_ACCESS_KEY")
	sessionToken, okSessionToken := os.LookupEnv("AWS_SESSION_TOKEN")
	defaultRegion, _ := os.LookupEnv("AWS_REGION")
	if defaultRegion == "" {
		defaultRegion = "us-west-1"
	}

	info := ""
	config := map[string]interface{}{}
	if !okAccessKeyID || accessKeyID == "" {
		info = "Skipping TestFetch; $AWS_ACCESS_KEY_ID not set or set to empty"
	} else if !okSecretAccessKey || secretAccessKey == "" {
		info = "Skipping TestFetch; $AWS_SECRET_ACCESS_KEY not set or set to empty"
	} else {
		config = map[string]interface{}{
			"module":            "aws",
			"period":            "300s",
			"metricsets":        []string{metricSetName},
			"access_key_id":     accessKeyID,
			"secret_access_key": secretAccessKey,
			"default_region":    defaultRegion,
		}

		if okSessionToken && sessionToken != "" {
			config["session_token"] = sessionToken
		}
	}
	return config, info
}

// CheckEventField function checks a given field type and compares it with the expected type for integration tests.
func CheckEventField(metricName string, expectedType string, event mb.Event, t *testing.T) {
	if ok, err := event.MetricSetFields.HasKey(metricName); ok {
		assert.NoError(t, err)
		metricValue, err := event.MetricSetFields.GetValue(metricName)
		assert.NoError(t, err)

		switch metricValue.(type) {
		case float64:
			if expectedType != "float" {
				t.Log("Failed: Field " + metricName + " is not in type " + expectedType)
				t.Fail()
			}
		case string:
			if expectedType != "string" {
				t.Log("Failed: Field " + metricName + " is not in type " + expectedType)
				t.Fail()
			}
		case int64:
			if expectedType != "int" {
				t.Log("Failed: Field " + metricName + " is not in type " + expectedType)
				t.Fail()
			}
		}
		t.Log("Succeed: Field " + metricName + " matches type " + expectedType)
	}
}
