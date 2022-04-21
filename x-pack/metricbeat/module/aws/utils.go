// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"math"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	resourcegroupstaggingapitypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
	"github.com/pkg/errors"
)

// GetStartTimeEndTime function uses durationString to create startTime and endTime for queries.
func GetStartTimeEndTime(period time.Duration, latency time.Duration) (time.Time, time.Time) {
	endTime := time.Now()
	if latency != 0 {
		// add latency if config is not 0
		endTime = endTime.Add(latency * -1)
	}

	// Set startTime to be one period earlier than the endTime. If metrics are
	// not being collected, use latency config parameter to offset the startTime
	// and endTime.
	startTime := endTime.Add(period * -1)
	// Defining duration
	d := 60 * time.Second
	// Calling Round() method
	return startTime.Round(d), endTime.Round(d)
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
			return metricsTotal, errors.Wrap(err, "error ListMetrics with Paginator, skipping region "+regionName)
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
				return getMetricDataOutput.MetricDataResults, errors.Wrap(err, "error GetMetricData with Paginator")
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
			err = errors.Wrap(err, "error GetResources with Paginator")
			return nil, err
		}

		for _, resourceTag := range page.ResourceTagMappingList {
			shortIdentifier, err := FindShortIdentifierFromARN(*resourceTag.ResourceARN)
			if err == nil {
				resourceTagMap[shortIdentifier] = resourceTag.Tags
			} else {
				err = errors.Wrap(err, "error occurs when processing shortIdentifier")
				return nil, err
			}

			wholeIdentifier, err := FindWholeIdentifierFromARN(*resourceTag.ResourceARN)
			if err == nil {
				resourceTagMap[wholeIdentifier] = resourceTag.Tags
			} else {
				err = errors.Wrap(err, "error occurs when processing longIdentifier")
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
		err = errors.Wrap(err, "error Parse arn")
		return "", err
	}

	resourceARNSplit := []string{arnParsed.Resource}
	if strings.Contains(arnParsed.Resource, ":") {
		resourceARNSplit = strings.Split(arnParsed.Resource, ":")
	} else if strings.Contains(arnParsed.Resource, "/") {
		resourceARNSplit = strings.Split(arnParsed.Resource, "/")
	}

	if len(resourceARNSplit) <= 1 {
		return resourceARNSplit[0], nil
	}
	return strings.Join(resourceARNSplit[1:], "/"), nil
}

// FindWholeIdentifierFromARN funtion extracts whole resource filed of ARN
func FindWholeIdentifierFromARN(resourceARN string) (string, error) {
	arnParsed, err := arn.Parse(resourceARN)
	if err != nil {
		err = errors.Wrap(err, "error Parse arn")
		return "", err
	}
	return arnParsed.Resource, nil
}
