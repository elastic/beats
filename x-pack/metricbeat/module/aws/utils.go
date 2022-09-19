// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	resourcegroupstaggingapitypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
)

// GetStartTimeEndTime calculates start and end times for queries based on the current time and a duration.
//
// Whilst the inputs to this function are continuous, the maximum period granularity we can consistently use
// is 1 minute. The resulting interval should also be aligned to the period for best performance. This means
// if a period of 3 minutes is requested at 12:05, for example, the calculated times are 12:00->12:03. See
// https://github.com/aws/aws-sdk-go-v2/blob/fdbd882cdf5c63a578caed14688cf9a456c75f2b/service/cloudwatch/api_op_GetMetricData.go#L88
// for more information about granularity and period alignment.
//
// If durations are configured in non-whole minute periods, they are rounded up to the next minute e.g. 90s becomes 120s.
//
// If `latency` is configured, the period is shifted back in time by specified duration (before period alignment).
func GetStartTimeEndTime(now time.Time, period time.Duration, latency time.Duration) (time.Time, time.Time) {
	periodInMinutes := (period + time.Second*29).Round(time.Second * 60)
	endTime := now.Add(latency * -1).Truncate(periodInMinutes)
	startTime := endTime.Add(periodInMinutes * -1)
	return startTime, endTime
}

// GetListMetricsOutput function gets listMetrics results from cloudwatch ~~per namespace~~ for each region.
// ListMetrics Cloudwatch API is used to list the specified metrics. The returned metrics can be used with GetMetricData
// to obtain statistical data.
func GetListMetricsOutput(namespace string, regionName string, svcCloudwatch cloudwatch.ListMetricsAPIClient) ([]types.Metric, error) {
	var metricsTotal []types.Metric
	var nextToken *string

	listMetricsInput := &cloudwatch.ListMetricsInput{
		NextToken: nextToken,
	}

	if namespace != "*" {
		listMetricsInput.Namespace = &namespace
	}

	paginator := cloudwatch.NewListMetricsPaginator(svcCloudwatch, listMetricsInput)

	// List metrics of a given namespace for each region
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return metricsTotal, fmt.Errorf("error ListMetrics with Paginator, skipping region %s: %w", regionName, err)
		}

		metricsTotal = append(metricsTotal, page.Metrics...)
	}
	return metricsTotal, nil
}

// GetMetricDataResults function uses MetricDataQueries to get metric data output.
func GetMetricDataResults(metricDataQueries []types.MetricDataQuery, svc cloudwatch.GetMetricDataAPIClient, startTime time.Time, endTime time.Time) ([]types.MetricDataResult, error) {
	maxQuerySize := 100
	getMetricDataOutput := &cloudwatch.GetMetricDataOutput{NextToken: nil}

	// Split metricDataQueries into smaller slices that length no longer than 100.
	// 100 is defined in maxQuerySize.
	// To avoid ValidationError: The collection MetricDataQueries must not have a size greater than 100.
	for i := 0; i < len(metricDataQueries); i += maxQuerySize {
		metricDataQueriesPartial := metricDataQueries[i:int(math.Min(float64(i+maxQuerySize), float64(len(metricDataQueries))))]
		if len(metricDataQueriesPartial) == 0 {
			return getMetricDataOutput.MetricDataResults, nil
		}

		getMetricDataInput := &cloudwatch.GetMetricDataInput{
			StartTime:         &startTime,
			EndTime:           &endTime,
			MetricDataQueries: metricDataQueriesPartial,
		}

		paginator := cloudwatch.NewGetMetricDataPaginator(svc, getMetricDataInput)
		var err error
		var page *cloudwatch.GetMetricDataOutput
		for paginator.HasMorePages() {
			if page, err = paginator.NextPage(context.TODO()); err != nil {
				return getMetricDataOutput.MetricDataResults, fmt.Errorf("error GetMetricData with Paginator: %w", err)
			}
			getMetricDataOutput.MetricDataResults = append(getMetricDataOutput.MetricDataResults, page.MetricDataResults...)
		}
	}

	return getMetricDataOutput.MetricDataResults, nil
}

// CheckTimestampInArray checks if input timestamp exists in timestampArray and if it exists, return the position.
func CheckTimestampInArray(timestamp time.Time, timestampArray []time.Time) (bool, int) {
	for i := 0; i < len(timestampArray); i++ {
		if timestamp.Equal(timestampArray[i]) {
			return true, i
		}
	}
	return false, -1
}

// FindTimestamp function checks MetricDataResults and find the timestamp to collect metrics from.
// For example, MetricDataResults might look like:
// metricDataResults =  [{
//	 Id: "sqs0",
//   Label: "testName SentMessageSize",
//   StatusCode: Complete,
//   Timestamps: [2019-03-11 17:45:00 +0000 UTC],
//   Values: [981]
// } {
//	 Id: "sqs1",
//	 Label: "testName NumberOfMessagesSent",
//	 StatusCode: Complete,
//	 Timestamps: [2019-03-11 17:45:00 +0000 UTC,2019-03-11 17:40:00 +0000 UTC],
//	 Values: [0.5,0]
// }]
// This case, we are collecting values for both metrics from timestamp 2019-03-11 17:45:00 +0000 UTC.
func FindTimestamp(getMetricDataResults []types.MetricDataResult) time.Time {
	timestamp := time.Time{}
	for _, output := range getMetricDataResults {
		// When there are outputs with one timestamp, use this timestamp.
		if output.Timestamps != nil && len(output.Timestamps) == 1 {
			// Use the first timestamp from Timestamps field to collect the latest data.
			timestamp = output.Timestamps[0]
			return timestamp
		}
	}

	// When there is no output with one timestamp, use the latest timestamp from timestamp list.
	if timestamp.IsZero() {
		for _, output := range getMetricDataResults {
			// When there are outputs with one timestamp, use this timestamp
			if output.Timestamps != nil && len(output.Timestamps) > 1 {
				// Example Timestamps: [2019-03-11 17:36:00 +0000 UTC,2019-03-11 17:31:00 +0000 UTC]
				timestamp = output.Timestamps[0]
				return timestamp
			}
		}
	}

	return timestamp
}

// GetResourcesTags function queries AWS resource groupings tagging API
// to get a resource tag mapping with specific resource type filters
func GetResourcesTags(svc resourcegroupstaggingapi.GetResourcesAPIClient, resourceTypeFilters []string) (map[string][]resourcegroupstaggingapitypes.Tag, error) {
	if resourceTypeFilters == nil {
		return map[string][]resourcegroupstaggingapitypes.Tag{}, nil
	}

	resourceTagMap := make(map[string][]resourcegroupstaggingapitypes.Tag)
	getResourcesInput := &resourcegroupstaggingapi.GetResourcesInput{
		PaginationToken:     nil,
		ResourceTypeFilters: resourceTypeFilters,
	}

	paginator := resourcegroupstaggingapi.NewGetResourcesPaginator(svc, getResourcesInput)
	var err error
	var page *resourcegroupstaggingapi.GetResourcesOutput
	for paginator.HasMorePages() {
		if page, err = paginator.NextPage(context.TODO()); err != nil {
			err = fmt.Errorf("error GetResources with Paginator: %w", err)
			return nil, err
		}

		for _, resourceTag := range page.ResourceTagMappingList {
			shortIdentifier, err := FindShortIdentifierFromARN(*resourceTag.ResourceARN)
			if err == nil {
				resourceTagMap[shortIdentifier] = resourceTag.Tags
			} else {
				err = fmt.Errorf("error occurs when processing shortIdentifier: %w", err)
				return nil, err
			}

			wholeIdentifier, err := FindWholeIdentifierFromARN(*resourceTag.ResourceARN)
			if err == nil {
				resourceTagMap[wholeIdentifier] = resourceTag.Tags
			} else {
				err = fmt.Errorf("error occurs when processing longIdentifier: %w", err)
				return nil, err
			}
		}
	}

	return resourceTagMap, nil
}

// FindShortIdentifierFromARN function extracts short resource id from resource filed of ARN.
func FindShortIdentifierFromARN(resourceARN string) (string, error) {
	arnParsed, err := arn.Parse(resourceARN)
	if err != nil {
		err = fmt.Errorf("error Parse arn: %w", err)
		return "", err
	}

	resourceARNSplit := []string{arnParsed.Resource}
	if strings.Contains(arnParsed.Resource, ":") {
		resourceARNSplit = strings.Split(strings.Trim(arnParsed.Resource, ":"), ":")
	} else if strings.Contains(arnParsed.Resource, "/") {
		resourceARNSplit = strings.Split(strings.Trim(arnParsed.Resource, "/"), "/")
	}

	if len(resourceARNSplit) <= 1 {
		return resourceARNSplit[0], nil
	}
	return strings.Join(resourceARNSplit[1:], "/"), nil
}

// FindWholeIdentifierFromARN function extracts whole resource filed of ARN
func FindWholeIdentifierFromARN(resourceARN string) (string, error) {
	arnParsed, err := arn.Parse(resourceARN)
	if err != nil {
		err = fmt.Errorf("error Parse arn: %w", err)
		return "", err
	}
	return arnParsed.Resource, nil
}
