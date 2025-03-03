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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	resourcegroupstaggingapitypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
)

const DefaultApiTimeout = 5 * time.Second

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
// If endTime of the previous collection period is recorded, then use this endTime as the new startTime. This will guarantee no gap between collection timestamps.
func GetStartTimeEndTime(now time.Time, period time.Duration, latency time.Duration, previousEndTime time.Time) (time.Time, time.Time) {
	periodInMinutes := (period + time.Second*29).Round(time.Second * 60)
	var startTime, endTime time.Time
	if !previousEndTime.IsZero() {
		startTime = previousEndTime
		endTime = startTime.Add(periodInMinutes)
	} else {
		endTime = now.Add(latency * -1).Truncate(periodInMinutes)
		startTime = endTime.Add(periodInMinutes * -1)
	}
	return startTime, endTime
}

// MetricWithID contains a specific metric, and its account ID information.
type MetricWithID struct {
	Metric    types.Metric
	AccountID string
}

// GetListMetricsOutput function gets listMetrics results from cloudwatch ~~per namespace~~ for each region.
// ListMetrics Cloudwatch API is used to list the specified metrics. The returned metrics can be used with GetMetricData
// to obtain statistical data.
// Note: We are not using Dimensions and MetricName in ListMetricsInput because with that we will have to make one ListMetrics
// API call per metric name and set of dimensions. This will increase API cost.
// IncludeLinkedAccounts is set to true for ListMetrics API to include metrics from source accounts in addition to the
// monitoring account.
// OwningAccount works alongside IncludeLinkedAccounts as a filter mechanism to extract metrics specific to a linked account.
func GetListMetricsOutput(namespace string, regionName string, period time.Duration, includeLinkedAccounts bool,
	owningAccount string, monitoringAccountID string, svcCloudwatch cloudwatch.ListMetricsAPIClient) ([]MetricWithID, error) {

	var metricWithAccountID []MetricWithID
	var nextToken *string

	listMetricsInput := &cloudwatch.ListMetricsInput{
		NextToken:             nextToken,
		IncludeLinkedAccounts: &includeLinkedAccounts,
	}

	if owningAccount != "" {
		listMetricsInput.OwningAccount = &owningAccount
	}

	// To filter the results to show only metrics that have had data points published
	// in the past three hours, specify this parameter with a value of PT3H.
	// Please see https://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/API_ListMetrics.html for more details.
	if period <= time.Hour*3 {
		listMetricsInput.RecentlyActive = types.RecentlyActivePt3h
	}

	if namespace != "*" {
		listMetricsInput.Namespace = &namespace
	}

	paginator := cloudwatch.NewListMetricsPaginator(svcCloudwatch, listMetricsInput)

	// List metrics of a given namespace for each region
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return metricWithAccountID, fmt.Errorf("error ListMetrics with Paginator, skipping region %s: %w", regionName, err)
		}

		for i, metric := range page.Metrics {
			if page.OwningAccounts == nil {
				// When IncludeLinkedAccounts is set to false, ListMetrics API does not return any OwningAccounts.
				// Hence, account ID is set to the monitoring account ID
				metricWithAccountID = append(metricWithAccountID, MetricWithID{metric, monitoringAccountID})
			} else {
				metricWithAccountID = append(metricWithAccountID, MetricWithID{metric, page.OwningAccounts[i]})
			}
		}
	}
	return metricWithAccountID, nil
}

// GetAPIGatewayRestAPIOutput function gets results from apigw api.
// GetRestApis Apigateway API is used to retrieve only the REST API specified info. This returns a map with the names and ids of RestAPIs configured
// Limit variable defines maximum number of returned results per page. The default value is 25 and the maximum value is 500.
func GetAPIGatewayRestAPIOutput(svcRestApi *apigateway.Client, limit *int32) (map[string]string, error) {
	input := &apigateway.GetRestApisInput{}
	if limit != nil {
		input = &apigateway.GetRestApisInput{
			Limit: limit,
		}
	}
	ctx, cancel := getContextWithTimeout(DefaultApiTimeout)
	defer cancel()
	result, err := svcRestApi.GetRestApis(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error retrieving GetRestApis %w", err)
	}

	// Iterate and display the APIs
	infoRestAPImap := make(map[string]string, len(result.Items))
	for _, api := range result.Items {
		infoRestAPImap[aws.ToString(api.Name)] = aws.ToString(api.Id)
	}
	return infoRestAPImap, nil
}

// GetAPIGatewayAPIOutput function gets results from apigatewayv2 api.
// GetApis Apigateway API is used to retrieve the HTTP and WEBSOCKET specified info. This returns a map with the names and ids of relevant APIs configured
func GetAPIGatewayAPIOutput(svcHttpApi *apigatewayv2.Client) (map[string]string, error) {
	input := &apigatewayv2.GetApisInput{}

	ctx, cancel := getContextWithTimeout(DefaultApiTimeout)
	defer cancel()
	result, err := svcHttpApi.GetApis(ctx, input)

	if err != nil {
		return nil, fmt.Errorf("error retrieving GetApis %w", err)
	}

	// Iterate and display the APIs
	infoAPImap := make(map[string]string, len(result.Items))
	for _, api := range result.Items {
		infoAPImap[aws.ToString(api.Name)] = aws.ToString(api.ApiId)
	}
	return infoAPImap, nil
}

// GetMetricDataResults function uses MetricDataQueries to get metric data output.
func GetMetricDataResults(metricDataQueries []types.MetricDataQuery, svc cloudwatch.GetMetricDataAPIClient, startTime time.Time, endTime time.Time) ([]types.MetricDataResult, error) {
	maxNumberOfMetricsRetrieved := 500
	getMetricDataOutput := &cloudwatch.GetMetricDataOutput{NextToken: nil}

	// Split metricDataQueries into smaller slices that length no longer than 500.
	// 500 is defined in maxNumberOfMetricsRetrieved.
	// To avoid ValidationError: The collection MetricDataQueries must not have a size greater than 500.
	for i := 0; i < len(metricDataQueries); i += maxNumberOfMetricsRetrieved {
		metricDataQueriesPartial := metricDataQueries[i:int(math.Min(float64(i+maxNumberOfMetricsRetrieved), float64(len(metricDataQueries))))]
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

func getContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
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
	ctx, cancel := getContextWithTimeout(DefaultApiTimeout)
	defer cancel()
	var err error
	var page *resourcegroupstaggingapi.GetResourcesOutput
	for paginator.HasMorePages() {
		if page, err = paginator.NextPage(ctx); err != nil {
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
