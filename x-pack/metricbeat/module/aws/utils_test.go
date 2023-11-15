// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cloudwatchtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	resourcegroupstaggingapitypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
	"github.com/stretchr/testify/assert"
)

var (
	metricName  = "CPUUtilization"
	namespace   = "AWS/EC2"
	dimName     = "InstanceId"
	instanceID1 = "i-123"
	instanceID2 = "i-456"

	id1    = "cpu1"
	label1 = instanceID1 + " " + metricName

	id2         = "status1"
	metricName2 = "StatusCheckFailed"
	label2      = instanceID1 + " " + metricName2

	id3         = "status2"
	metricName3 = "StatusCheckFailed_System"
	label3      = instanceID1 + " " + metricName3

	id4         = "status3"
	metricName4 = "StatusCheckFailed_Instance"
	label4      = instanceID1 + " " + metricName4

	id5    = "cpu2"
	label5 = instanceID2 + " " + metricName
)

// MockCloudwatchClient struct is used for unit tests.
type MockCloudWatchClient struct{}

// MockCloudwatchClientCrossAccounts struct is used for unit tests.
type MockCloudwatchClientCrossAccounts struct{}

// GetMetricData implements cloudwatch.GetMetricDataAPIClient interface
func (m *MockCloudWatchClient) GetMetricData(context.Context, *cloudwatch.GetMetricDataInput, ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
	emptyString := ""
	value1 := 0.25
	value2 := 0.0
	value3 := 0.0
	value4 := 0.0

	return &cloudwatch.GetMetricDataOutput{
		Messages: nil,
		MetricDataResults: []cloudwatchtypes.MetricDataResult{
			{
				Id:     &id1,
				Label:  &label1,
				Values: []float64{value1},
			},
			{
				Id:     &id2,
				Label:  &label2,
				Values: []float64{value2},
			},
			{
				Id:     &id3,
				Label:  &label3,
				Values: []float64{value3},
			},
			{
				Id:     &id4,
				Label:  &label4,
				Values: []float64{value4},
			},
		},
		NextToken: &emptyString,
	}, nil
}

// GetMetricData implements cloudwatch.GetMetricDataAPIClient interface for cross accounts
func (m *MockCloudwatchClientCrossAccounts) GetMetricData(context.Context, *cloudwatch.GetMetricDataInput, ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
	emptyString := ""
	value1 := 0.25
	value2 := 0.15

	return &cloudwatch.GetMetricDataOutput{
		Messages: nil,
		MetricDataResults: []cloudwatchtypes.MetricDataResult{
			{
				Id:     &id1,
				Label:  &label1,
				Values: []float64{value1},
			},
			{
				Id:     &id5,
				Label:  &label5,
				Values: []float64{value2},
			},
		},
		NextToken: &emptyString,
	}, nil
}

func (m *MockCloudWatchClient) ListMetrics(context.Context, *cloudwatch.ListMetricsInput, ...func(*cloudwatch.Options)) (*cloudwatch.ListMetricsOutput, error) {
	dim1 := cloudwatchtypes.Dimension{
		Name:  &dimName,
		Value: &instanceID1,
	}

	return &cloudwatch.ListMetricsOutput{
		Metrics: []cloudwatchtypes.Metric{
			{
				MetricName: &metricName,
				Namespace:  &namespace,
				Dimensions: []cloudwatchtypes.Dimension{dim1},
			},
		},
		NextToken: awssdk.String(""),
	}, nil
}

func (m *MockCloudwatchClientCrossAccounts) ListMetrics(context.Context, *cloudwatch.ListMetricsInput, ...func(*cloudwatch.Options)) (*cloudwatch.ListMetricsOutput, error) {
	dim1 := cloudwatchtypes.Dimension{
		Name:  &dimName,
		Value: &instanceID1,
	}
	dim2 := cloudwatchtypes.Dimension{
		Name:  &dimName,
		Value: &instanceID2,
	}

	return &cloudwatch.ListMetricsOutput{
		Metrics: []cloudwatchtypes.Metric{
			{
				MetricName: &metricName,
				Namespace:  &namespace,
				Dimensions: []cloudwatchtypes.Dimension{dim1},
			},
			{
				MetricName: &metricName,
				Namespace:  &namespace,
				Dimensions: []cloudwatchtypes.Dimension{dim2},
			},
		},
		OwningAccounts: []string{
			"123",
			"456",
		},
		NextToken: awssdk.String(""),
	}, nil
}

// MockResourceGroupsTaggingClient is used for unit tests.
type MockResourceGroupsTaggingClient struct{}

// GetResources implements resourcegroupstaggingapi.GetResourcesAPIClient.
func (m *MockResourceGroupsTaggingClient) GetResources(_ context.Context, _ *resourcegroupstaggingapi.GetResourcesInput, _ ...func(*resourcegroupstaggingapi.Options)) (*resourcegroupstaggingapi.GetResourcesOutput, error) {
	return &resourcegroupstaggingapi.GetResourcesOutput{
		PaginationToken: awssdk.String(""),
		ResourceTagMappingList: []resourcegroupstaggingapitypes.ResourceTagMapping{
			{
				ResourceARN: awssdk.String("arn:aws:rds:eu-west-1:123456789012:db:mysql-db-1"),
				Tags: []resourcegroupstaggingapitypes.Tag{
					{
						Key:   awssdk.String("organization"),
						Value: awssdk.String("engineering"),
					},
					{
						Key:   awssdk.String("owner"),
						Value: awssdk.String("foo"),
					},
				},
			},
			{
				ResourceARN: awssdk.String("arn:aws:rds:eu-west-1:123456789012:db:mysql-db-2"),
				Tags: []resourcegroupstaggingapitypes.Tag{
					{
						Key:   awssdk.String("organization"),
						Value: awssdk.String("finance"),
					},
					{
						Key:   awssdk.String("owner"),
						Value: awssdk.String("boo"),
					},
				},
			},
		},
	}, nil
}

func TestGetListMetricsOutput(t *testing.T) {
	svcCloudwatch := &MockCloudWatchClient{}
	listMetricsOutput, err := GetListMetricsOutput("AWS/EC2", "us-west-1", time.Minute*5, false, "123", svcCloudwatch)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(listMetricsOutput))
	assert.Equal(t, namespace, *listMetricsOutput[0].Metric.Namespace)
	assert.Equal(t, metricName, *listMetricsOutput[0].Metric.MetricName)
	assert.Equal(t, 1, len(listMetricsOutput[0].Metric.Dimensions))
	assert.Equal(t, dimName, *listMetricsOutput[0].Metric.Dimensions[0].Name)
	assert.Equal(t, instanceID1, *listMetricsOutput[0].Metric.Dimensions[0].Value)
}

func TestGetListMetricsCrossAccountsOutput(t *testing.T) {
	svcCloudwatch := &MockCloudwatchClientCrossAccounts{}
	listMetricsOutput, err := GetListMetricsOutput("AWS/EC2", "us-west-1", time.Minute*5, true, "123", svcCloudwatch)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(listMetricsOutput))
	assert.Equal(t, namespace, *listMetricsOutput[0].Metric.Namespace)
	assert.Equal(t, metricName, *listMetricsOutput[0].Metric.MetricName)
	assert.Equal(t, 1, len(listMetricsOutput[0].Metric.Dimensions))
	assert.Equal(t, dimName, *listMetricsOutput[0].Metric.Dimensions[0].Name)
	assert.Equal(t, instanceID1, *listMetricsOutput[0].Metric.Dimensions[0].Value)
	assert.Equal(t, instanceID2, *listMetricsOutput[1].Metric.Dimensions[0].Value)
}

func TestGetListMetricsOutputWithWildcard(t *testing.T) {
	svcCloudwatch := &MockCloudWatchClient{}
	listMetricsOutput, err := GetListMetricsOutput("*", "us-west-1", time.Minute*5, false, "123", svcCloudwatch)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(listMetricsOutput))
	assert.Equal(t, namespace, *listMetricsOutput[0].Metric.Namespace)
	assert.Equal(t, metricName, *listMetricsOutput[0].Metric.MetricName)
	assert.Equal(t, 1, len(listMetricsOutput[0].Metric.Dimensions))
	assert.Equal(t, dimName, *listMetricsOutput[0].Metric.Dimensions[0].Name)
	assert.Equal(t, instanceID1, *listMetricsOutput[0].Metric.Dimensions[0].Value)
}

func TestGetMetricDataPerRegion(t *testing.T) {
	startTime, endTime := GetStartTimeEndTime(time.Now(), 10*time.Minute, 0)

	mockSvc := &MockCloudWatchClient{}
	var metricDataQueries []cloudwatchtypes.MetricDataQuery

	getMetricDataInput := &cloudwatch.GetMetricDataInput{
		NextToken:         nil,
		StartTime:         &startTime,
		EndTime:           &endTime,
		MetricDataQueries: metricDataQueries,
	}

	getMetricDataOutput, err := mockSvc.GetMetricData(context.TODO(), getMetricDataInput)
	assert.NoError(t, err)

	assert.Equal(t, 4, len(getMetricDataOutput.MetricDataResults))
	assert.Equal(t, id1, *getMetricDataOutput.MetricDataResults[0].Id)
	assert.Equal(t, label1, *getMetricDataOutput.MetricDataResults[0].Label)
	assert.Equal(t, 0.25, getMetricDataOutput.MetricDataResults[0].Values[0])

	assert.Equal(t, id2, *getMetricDataOutput.MetricDataResults[1].Id)
	assert.Equal(t, label2, *getMetricDataOutput.MetricDataResults[1].Label)
	assert.Equal(t, 0.0, getMetricDataOutput.MetricDataResults[1].Values[0])

	assert.Equal(t, id3, *getMetricDataOutput.MetricDataResults[2].Id)
	assert.Equal(t, label3, *getMetricDataOutput.MetricDataResults[2].Label)
	assert.Equal(t, 0.0, getMetricDataOutput.MetricDataResults[2].Values[0])

	assert.Equal(t, id4, *getMetricDataOutput.MetricDataResults[3].Id)
	assert.Equal(t, label4, *getMetricDataOutput.MetricDataResults[3].Label)
	assert.Equal(t, 0.0, getMetricDataOutput.MetricDataResults[3].Values[0])
}

func TestGetMetricDataResults(t *testing.T) {
	startTime, endTime := GetStartTimeEndTime(time.Now(), 10*time.Minute, 0)

	mockSvc := &MockCloudWatchClient{}
	metricInfo := cloudwatchtypes.Metric{
		MetricName: &metricName,
		Namespace:  &namespace,
	}
	metricStat := cloudwatchtypes.MetricStat{Metric: &metricInfo}
	metricDataQueries := []cloudwatchtypes.MetricDataQuery{
		{
			Id:         &id1,
			Label:      &label1,
			MetricStat: &metricStat,
		},
	}
	getMetricDataResults, err := GetMetricDataResults(metricDataQueries, mockSvc, startTime, endTime)
	assert.NoError(t, err)

	assert.Equal(t, 4, len(getMetricDataResults))
	assert.Equal(t, id1, *getMetricDataResults[0].Id)
	assert.Equal(t, label1, *getMetricDataResults[0].Label)
	assert.Equal(t, 0.25, getMetricDataResults[0].Values[0])

	assert.Equal(t, id2, *getMetricDataResults[1].Id)
	assert.Equal(t, label2, *getMetricDataResults[1].Label)
	assert.Equal(t, 0.0, getMetricDataResults[1].Values[0])

	assert.Equal(t, id3, *getMetricDataResults[2].Id)
	assert.Equal(t, label3, *getMetricDataResults[2].Label)
	assert.Equal(t, 0.0, getMetricDataResults[2].Values[0])

	assert.Equal(t, id4, *getMetricDataResults[3].Id)
	assert.Equal(t, label4, *getMetricDataResults[3].Label)
	assert.Equal(t, 0.0, getMetricDataResults[3].Values[0])
}

func TestGetMetricDataResultsCrossAccounts(t *testing.T) {
	startTime, endTime := GetStartTimeEndTime(time.Now(), 10*time.Minute, 0)

	mockSvc := &MockCloudwatchClientCrossAccounts{}
	metricInfo := cloudwatchtypes.Metric{
		MetricName: &metricName,
		Namespace:  &namespace,
	}
	metricStat := cloudwatchtypes.MetricStat{Metric: &metricInfo}
	metricDataQueries := []cloudwatchtypes.MetricDataQuery{
		{
			Id:         &id1,
			Label:      &label1,
			MetricStat: &metricStat,
		},
	}
	getMetricDataResults, err := GetMetricDataResults(metricDataQueries, mockSvc, startTime, endTime)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(getMetricDataResults))
	assert.Equal(t, id1, *getMetricDataResults[0].Id)
	assert.Equal(t, label1, *getMetricDataResults[0].Label)
	assert.Equal(t, 0.25, getMetricDataResults[0].Values[0])

	assert.Equal(t, id5, *getMetricDataResults[1].Id)
	assert.Equal(t, label5, *getMetricDataResults[1].Label)
	assert.Equal(t, 0.15, getMetricDataResults[1].Values[0])
}

func TestCheckTimestampInArray(t *testing.T) {
	timestamp1 := time.Now()
	timestamp2 := timestamp1.Add(5 * time.Minute)
	timestamp3 := timestamp1.Add(10 * time.Minute)

	cases := []struct {
		targetTimestamp time.Time
		expectedExists  bool
		expectedIndex   int
	}{
		{
			targetTimestamp: timestamp1,
			expectedExists:  true,
			expectedIndex:   0,
		},
		{
			targetTimestamp: timestamp3,
			expectedExists:  false,
			expectedIndex:   -1,
		},
	}

	timestampArray := []time.Time{timestamp1, timestamp2}
	for _, c := range cases {
		exists, index := CheckTimestampInArray(c.targetTimestamp, timestampArray)
		assert.Equal(t, c.expectedExists, exists)
		assert.Equal(t, c.expectedIndex, index)
	}
}

func TestFindIdentifierFromARN(t *testing.T) {
	cases := []struct {
		resourceARN             string
		expectedShortIdentifier string
		expectedWholeIdentifier string
	}{
		{
			"arn:aws:rds:eu-west-1:123456789012:db:mysql-db",
			"mysql-db",
			"db:mysql-db",
		},
		{
			"arn:aws:ec2:us-east-1:123456789012:instance/i-123",
			"i-123",
			"instance/i-123",
		},
		{
			"arn:aws:sns:us-east-1:627959692251:notification-topic-1",
			"notification-topic-1",
			"notification-topic-1",
		},
		{
			"arn:aws:elasticloadbalancing:eu-central-1:627959692251:loadbalancer/app/ece-ui/b195d6cf21493989",
			"app/ece-ui/b195d6cf21493989",
			"loadbalancer/app/ece-ui/b195d6cf21493989",
		},
		{
			"arn:aws:elasticloadbalancing:eu-central-1:627959692251:loadbalancer/net/ece-es-clusters-nlb/0c5bdb3b96cf1552",
			"net/ece-es-clusters-nlb/0c5bdb3b96cf1552",
			"loadbalancer/net/ece-es-clusters-nlb/0c5bdb3b96cf1552",
		},
		{
			"arn:aws:apigateway:us-east-1::/apis/lqyipneb7c",
			"lqyipneb7c",
			"/apis/lqyipneb7c",
		},
	}

	for _, c := range cases {
		shortIdentifier, err := FindShortIdentifierFromARN(c.resourceARN)
		assert.NoError(t, err)
		assert.Equal(t, c.expectedShortIdentifier, shortIdentifier)

		wholeIdentifier, err := FindWholeIdentifierFromARN(c.resourceARN)
		assert.NoError(t, err)
		assert.Equal(t, c.expectedWholeIdentifier, wholeIdentifier)
	}

}

func TestGetResourcesTags(t *testing.T) {
	mockSvc := &MockResourceGroupsTaggingClient{}
	resourceTagMap, err := GetResourcesTags(mockSvc, []string{"rds"})
	assert.NoError(t, err)
	assert.Equal(t, 4, len(resourceTagMap))

	expectedResourceTagMap := map[string][]resourcegroupstaggingapitypes.Tag{}
	expectedResourceTagMap["mysql-db-1"] = []resourcegroupstaggingapitypes.Tag{
		{
			Key:   awssdk.String("organization"),
			Value: awssdk.String("engineering"),
		},
		{
			Key:   awssdk.String("owner"),
			Value: awssdk.String("foo"),
		},
	}
	expectedResourceTagMap["mysql-db-2"] = []resourcegroupstaggingapitypes.Tag{
		{
			Key:   awssdk.String("organization"),
			Value: awssdk.String("finance"),
		},
		{
			Key:   awssdk.String("owner"),
			Value: awssdk.String("boo"),
		},
	}
	expectedResourceTagMap["db:mysql-db-1"] = []resourcegroupstaggingapitypes.Tag{
		{
			Key:   awssdk.String("organization"),
			Value: awssdk.String("engineering"),
		},
		{
			Key:   awssdk.String("owner"),
			Value: awssdk.String("foo"),
		},
	}
	expectedResourceTagMap["db:mysql-db-2"] = []resourcegroupstaggingapitypes.Tag{
		{
			Key:   awssdk.String("organization"),
			Value: awssdk.String("finance"),
		},
		{
			Key:   awssdk.String("owner"),
			Value: awssdk.String("boo"),
		},
	}
	assert.Equal(t, expectedResourceTagMap, resourceTagMap)
}

func parseTime(t *testing.T, in string) time.Time {
	time, err := time.Parse(time.RFC3339, in)
	if err != nil {
		t.Errorf("test setup failed - could not parse time with time.RFC3339: %s", in)
	}
	return time
}

func TestGetStartTimeEndTime(t *testing.T) {
	var cases = []struct {
		title         string
		start         string
		period        time.Duration
		latency       time.Duration
		expectedStart string
		expectedEnd   string
	}{
		// window should align with period e.g. requesting a 5 minute period at 10:27 gives 10:20->10:25
		{"1 minute", "2022-08-15T13:38:45Z", time.Second * 60, 0, "2022-08-15T13:37:00Z", "2022-08-15T13:38:00Z"},
		{"2 minutes", "2022-08-15T13:38:45Z", time.Second * 60 * 2, 0, "2022-08-15T13:36:00Z", "2022-08-15T13:38:00Z"},
		{"3 minutes", "2022-08-15T13:38:45Z", time.Second * 60 * 3, 0, "2022-08-15T13:33:00Z", "2022-08-15T13:36:00Z"},
		{"5 minutes", "2022-08-15T13:38:45Z", time.Second * 60 * 5, 0, "2022-08-15T13:30:00Z", "2022-08-15T13:35:00Z"},
		{"30 minutes", "2022-08-15T13:38:45Z", time.Second * 60 * 30, 0, "2022-08-15T13:00:00Z", "2022-08-15T13:30:00Z"},

		// latency should shift the time *before* period alignment
		// e.g. requesting a 5 minute period at 10:27 with 1 minutes latency still gives 10:20->10:25,
		//      but with 3 minutes latency gives 10:15->10:20
		{"1 minute, 10 minutes latency", "2022-08-15T13:38:45Z", time.Second * 60, time.Second * 60 * 10, "2022-08-15T13:27:00Z", "2022-08-15T13:28:00Z"},
		{"2 minutes, 1 minute latency", "2022-08-15T13:38:45Z", time.Second * 60 * 2, time.Second * 60, "2022-08-15T13:34:00Z", "2022-08-15T13:36:00Z"},
		{"5 minutes, 4 minutes latency", "2022-08-15T13:38:45Z", time.Second * 60 * 5, time.Second * 60 * 4, "2022-08-15T13:25:00Z", "2022-08-15T13:30:00Z"},
		{"30 minutes, 30 minutes latency", "2022-08-15T13:38:45Z", time.Second * 60 * 30, time.Second * 60 * 30, "2022-08-15T12:30:00Z", "2022-08-15T13:00:00Z"},

		// non-whole-minute periods should be rounded up to the nearest minute; latency is applied as-is before period adjustment
		{"20 seconds, 45 second latency", "2022-08-15T13:38:45Z", time.Second * 20, time.Second * 45, "2022-08-15T13:37:00Z", "2022-08-15T13:38:00Z"},
		{"1.5 minutes, 60 second latency", "2022-08-15T13:38:45Z", time.Second * 90, time.Second * 60, "2022-08-15T13:34:00Z", "2022-08-15T13:36:00Z"},
		{"just less than 5 minutes, 3 minute latency", "2022-08-15T13:38:45Z", time.Second * 59 * 5, time.Second * 90, "2022-08-15T13:30:00Z", "2022-08-15T13:35:00Z"},
	}

	for _, tt := range cases {
		t.Run(tt.title, func(t *testing.T) {
			startTime, expectedStartTime, expectedEndTime := parseTime(t, tt.start), parseTime(t, tt.expectedStart), parseTime(t, tt.expectedEnd)

			start, end := GetStartTimeEndTime(startTime, tt.period, tt.latency)

			if expectedStartTime != start || expectedEndTime != end {
				t.Errorf("got (%s, %s), want (%s, %s)", start, end, tt.expectedStart, tt.expectedEnd)
			}
		})
	}
}

func TestGetStartTimeEndTime_AlwaysCreatesContinuousIntervals(t *testing.T) {
	type interval struct {
		start, end string
	}

	startTime := parseTime(t, "2022-08-24T11:01:00Z")
	numCalls := 5

	var cases = []struct {
		title             string
		period            time.Duration
		latency           time.Duration
		expectedIntervals []interval
	}{
		// with no latency
		{"1 minute", time.Second * 60, 0, []interval{
			{"2022-08-24T11:00:00Z", "2022-08-24T11:01:00Z"},
			{"2022-08-24T11:01:00Z", "2022-08-24T11:02:00Z"},
			{"2022-08-24T11:02:00Z", "2022-08-24T11:03:00Z"},
			{"2022-08-24T11:03:00Z", "2022-08-24T11:04:00Z"},
			{"2022-08-24T11:04:00Z", "2022-08-24T11:05:00Z"},
		}},
		{"2 minutes", time.Second * 60 * 2, 0, []interval{
			{"2022-08-24T10:58:00Z", "2022-08-24T11:00:00Z"},
			{"2022-08-24T11:00:00Z", "2022-08-24T11:02:00Z"},
			{"2022-08-24T11:02:00Z", "2022-08-24T11:04:00Z"},
			{"2022-08-24T11:04:00Z", "2022-08-24T11:06:00Z"},
			{"2022-08-24T11:06:00Z", "2022-08-24T11:08:00Z"},
		}},
		{"3 minutes", time.Second * 60 * 3, 0, []interval{
			{"2022-08-24T10:57:00Z", "2022-08-24T11:00:00Z"},
			{"2022-08-24T11:00:00Z", "2022-08-24T11:03:00Z"},
			{"2022-08-24T11:03:00Z", "2022-08-24T11:06:00Z"},
			{"2022-08-24T11:06:00Z", "2022-08-24T11:09:00Z"},
			{"2022-08-24T11:09:00Z", "2022-08-24T11:12:00Z"},
		}},
		{"5 minutes", time.Second * 60 * 5, 0, []interval{
			{"2022-08-24T10:55:00Z", "2022-08-24T11:00:00Z"},
			{"2022-08-24T11:00:00Z", "2022-08-24T11:05:00Z"},
			{"2022-08-24T11:05:00Z", "2022-08-24T11:10:00Z"},
			{"2022-08-24T11:10:00Z", "2022-08-24T11:15:00Z"},
			{"2022-08-24T11:15:00Z", "2022-08-24T11:20:00Z"},
		}},
		{"30 minutes", time.Second * 60 * 30, 0, []interval{
			{"2022-08-24T10:30:00Z", "2022-08-24T11:00:00Z"},
			{"2022-08-24T11:00:00Z", "2022-08-24T11:30:00Z"},
			{"2022-08-24T11:30:00Z", "2022-08-24T12:00:00Z"},
			{"2022-08-24T12:00:00Z", "2022-08-24T12:30:00Z"},
			{"2022-08-24T12:30:00Z", "2022-08-24T13:00:00Z"},
		}},

		// with 90s latency (sanity check)
		{"1 minute with 2 minute latency", time.Second * 60, time.Second * 90, []interval{
			{"2022-08-24T10:58:00Z", "2022-08-24T10:59:00Z"},
			{"2022-08-24T10:59:00Z", "2022-08-24T11:00:00Z"},
			{"2022-08-24T11:00:00Z", "2022-08-24T11:01:00Z"},
			{"2022-08-24T11:01:00Z", "2022-08-24T11:02:00Z"},
			{"2022-08-24T11:02:00Z", "2022-08-24T11:03:00Z"},
		}},
	}

	for _, tt := range cases {
		t.Run(tt.title, func(t *testing.T) {
			// get a few repeated intervals
			intervals := make([]interval, numCalls)
			for i := range intervals {
				adjustedStartTime := startTime.Add(tt.period * time.Duration(i))
				start, end := GetStartTimeEndTime(adjustedStartTime, tt.period, tt.latency)
				intervals[i] = interval{start.Format(time.RFC3339), end.Format(time.RFC3339)}
			}

			for i, val := range intervals {
				if val != tt.expectedIntervals[i] {
					t.Errorf("got %v, want %v", intervals, tt.expectedIntervals)
					break
				}
			}
		})
	}
}
