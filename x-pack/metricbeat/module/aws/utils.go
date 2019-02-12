// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"os"
	"time"
	"github.com/elastic/beats/libbeat/common"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/cloudwatchiface"
	"github.com/pkg/errors"
	s "github.com/elastic/beats/libbeat/common/schema"
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

// GetMetricDataPerRegion function uses MetricDataQueries to get metric data output.
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
func GetMetricDataResults(metricDataQueries []cloudwatch.MetricDataQuery, svc cloudwatchiface.CloudWatchAPI, startTime time.Time, endTime time.Time, regionName string) ([]cloudwatch.MetricDataResult, error) {
	init := true
	getMetricDataOutput := &cloudwatch.GetMetricDataOutput{NextToken: nil}
	for init || getMetricDataOutput.NextToken != nil {
		init = false
		output, err := getMetricDataPerRegion(metricDataQueries, getMetricDataOutput.NextToken, svc, startTime, endTime)
		if err != nil {
			err = errors.Wrap(err, "getMetricDataPerRegion failed, skipping region "+regionName)
			return getMetricDataOutput.MetricDataResults, err
		}
		getMetricDataOutput.MetricDataResults = append(getMetricDataOutput.MetricDataResults, output.MetricDataResults...)
	}
	return getMetricDataOutput.MetricDataResults, nil
}

// GetListMetricsOutput function gets listMetrics results from cloudwatch per namespace for each region.
// ListMetrics Cloudwatch API is used to list the specified metrics. The returned metrics can be used with GetMetricData
// to obtain statistical data.
func GetListMetricsOutput(namespace string, regionName string, svcCloudwatch *cloudwatch.CloudWatch) ([]cloudwatch.Metric, error) {
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
