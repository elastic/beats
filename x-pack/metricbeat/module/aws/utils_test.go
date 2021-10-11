// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"github.com/stretchr/testify/assert"
)

// MockCloudwatchClient struct is used for unit tests.
type MockCloudWatchClient struct {
	cloudwatchiface.ClientAPI
}

// MockResourceGroupsTaggingClient is used for unit tests.
type MockResourceGroupsTaggingClient struct {
	resourcegroupstaggingapiiface.ClientAPI
}

var (
	metricName = "CPUUtilization"
	namespace  = "AWS/EC2"
	dimName    = "InstanceId"
	instanceID = "i-123"

	id1         = "cpu1"
	metricName1 = "CPUUtilization"
	label1      = instanceID + " " + metricName1

	id2         = "status1"
	metricName2 = "StatusCheckFailed"
	label2      = instanceID + " " + metricName2

	id3         = "status2"
	metricName3 = "StatusCheckFailed_System"
	label3      = instanceID + " " + metricName3

	id4         = "status3"
	metricName4 = "StatusCheckFailed_Instance"
	label4      = instanceID + " " + metricName4
)

func (m *MockCloudWatchClient) ListMetricsRequest(input *cloudwatch.ListMetricsInput) cloudwatch.ListMetricsRequest {
	dim := cloudwatch.Dimension{
		Name:  &dimName,
		Value: &instanceID,
	}
	httpReq, _ := http.NewRequest("", "", nil)
	return cloudwatch.ListMetricsRequest{
		Input: input,
		Copy:  m.ListMetricsRequest,
		Request: &awssdk.Request{
			Operation: &awssdk.Operation{
				Name:      "ListMetrics",
				Paginator: nil,
			},
			Data: &cloudwatch.ListMetricsOutput{
				Metrics: []cloudwatch.Metric{
					{
						MetricName: &metricName,
						Namespace:  &namespace,
						Dimensions: []cloudwatch.Dimension{dim},
					},
				},
			},
			HTTPRequest: httpReq,
			Retryer:     awssdk.NoOpRetryer{},
		},
	}
}

func (m *MockCloudWatchClient) GetMetricDataRequest(input *cloudwatch.GetMetricDataInput) cloudwatch.GetMetricDataRequest {
	value1 := 0.25
	value2 := 0.0
	value3 := 0.0
	value4 := 0.0
	httpReq, _ := http.NewRequest("", "", nil)

	return cloudwatch.GetMetricDataRequest{
		Input: input,
		Copy:  m.GetMetricDataRequest,
		Request: &awssdk.Request{
			Operation: &awssdk.Operation{
				Name:      "GetMetricData",
				Paginator: nil,
			},
			Data: &cloudwatch.GetMetricDataOutput{
				MetricDataResults: []cloudwatch.MetricDataResult{
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
			},
			HTTPRequest: httpReq,
			Retryer:     awssdk.NoOpRetryer{},
		},
	}
}

func (m *MockResourceGroupsTaggingClient) GetResourcesRequest(input *resourcegroupstaggingapi.GetResourcesInput) resourcegroupstaggingapi.GetResourcesRequest {
	httpReq, _ := http.NewRequest("", "", nil)
	op := &awssdk.Operation{
		Name:       "GetResources",
		HTTPMethod: "POST",
		HTTPPath:   "/",
		Paginator:  nil,
	}
	firstPageResult := resourcegroupstaggingapi.GetResourcesRequest{
		Request: &awssdk.Request{
			Operation: op,
			Data: &resourcegroupstaggingapi.GetResourcesOutput{
				PaginationToken: awssdk.String("PaginationToken"),
				ResourceTagMappingList: []resourcegroupstaggingapi.ResourceTagMapping{
					{
						ResourceARN: awssdk.String("arn:aws:rds:eu-west-1:123456789012:db:mysql-db-1"),
						Tags: []resourcegroupstaggingapi.Tag{
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
						Tags: []resourcegroupstaggingapi.Tag{
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
			},
			HTTPRequest: httpReq,
			Retryer:     awssdk.NoOpRetryer{},
		},
		Input: input,
		Copy:  m.GetResourcesRequest,
	}

	// aws resourcegroupstaggingapi default pagination size is 50, if resource amount is a
	// multiple of 50, then last request has an empty result.
	lastPageWithEmptyResult := resourcegroupstaggingapi.GetResourcesRequest{
		Request: &awssdk.Request{
			Data: &resourcegroupstaggingapi.GetResourcesOutput{
				PaginationToken:        awssdk.String(""),
				ResourceTagMappingList: []resourcegroupstaggingapi.ResourceTagMapping{},
			},
			HTTPRequest: httpReq,
			Operation:   op,
			Retryer:     awssdk.NoOpRetryer{},
		},
		Input: input,
		Copy:  m.GetResourcesRequest,
	}

	if input.PaginationToken == nil {
		return firstPageResult
	} else {
		return lastPageWithEmptyResult
	}
}

func TestGetListMetricsOutput(t *testing.T) {
	svcCloudwatch := &MockCloudWatchClient{}
	listMetricsOutput, err := GetListMetricsOutput("AWS/EC2", "us-west-1", svcCloudwatch)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(listMetricsOutput))
	assert.Equal(t, namespace, *listMetricsOutput[0].Namespace)
	assert.Equal(t, metricName, *listMetricsOutput[0].MetricName)
	assert.Equal(t, 1, len(listMetricsOutput[0].Dimensions))
	assert.Equal(t, dimName, *listMetricsOutput[0].Dimensions[0].Name)
	assert.Equal(t, instanceID, *listMetricsOutput[0].Dimensions[0].Value)
}

func TestGetListMetricsOutputWithWildcard(t *testing.T) {
	svcCloudwatch := &MockCloudWatchClient{}
	listMetricsOutput, err := GetListMetricsOutput("*", "us-west-1", svcCloudwatch)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(listMetricsOutput))
	assert.Equal(t, namespace, *listMetricsOutput[0].Namespace)
	assert.Equal(t, metricName, *listMetricsOutput[0].MetricName)
	assert.Equal(t, 1, len(listMetricsOutput[0].Dimensions))
	assert.Equal(t, dimName, *listMetricsOutput[0].Dimensions[0].Name)
	assert.Equal(t, instanceID, *listMetricsOutput[0].Dimensions[0].Value)
}

func TestGetMetricDataPerRegion(t *testing.T) {
	startTime, endTime := GetStartTimeEndTime(10*time.Minute, 0)

	mockSvc := &MockCloudWatchClient{}
	var metricDataQueries []cloudwatch.MetricDataQuery

	getMetricDataInput := &cloudwatch.GetMetricDataInput{
		NextToken:         nil,
		StartTime:         &startTime,
		EndTime:           &endTime,
		MetricDataQueries: metricDataQueries,
	}

	reqGetMetricData := mockSvc.GetMetricDataRequest(getMetricDataInput)
	getMetricDataOutput, err := reqGetMetricData.Send(context.TODO())
	if err != nil {
		fmt.Println("failed getMetricDataPerRegion: ", err)
		t.FailNow()
	}

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
	startTime, endTime := GetStartTimeEndTime(10*time.Minute, 0)

	mockSvc := &MockCloudWatchClient{}
	metricInfo := cloudwatch.Metric{
		MetricName: &metricName,
		Namespace:  &namespace,
	}
	metricStat := cloudwatch.MetricStat{Metric: &metricInfo}
	metricDataQueries := []cloudwatch.MetricDataQuery{
		{
			Id:         &id1,
			Label:      &label1,
			MetricStat: &metricStat,
		},
	}
	getMetricDataResults, err := GetMetricDataResults(metricDataQueries, mockSvc, startTime, endTime)
	if err != nil {
		fmt.Println("failed getMetricDataPerRegion: ", err)
		t.FailNow()
	}

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

func TestFindTimestamp(t *testing.T) {
	timestamp1 := time.Now()
	timestamp2 := timestamp1.Add(5 * time.Minute)
	cases := []struct {
		getMetricDataResults []cloudwatch.MetricDataResult
		expectedTimestamp    time.Time
	}{
		{
			getMetricDataResults: []cloudwatch.MetricDataResult{
				{
					Id:         &id1,
					Label:      &label1,
					StatusCode: cloudwatch.StatusCodeComplete,
					Timestamps: []time.Time{timestamp1, timestamp2},
					Values:     []float64{0, 1},
				},
				{
					Id:         &id2,
					Label:      &label2,
					StatusCode: cloudwatch.StatusCodeComplete,
					Timestamps: []time.Time{timestamp1},
					Values:     []float64{2, 3},
				},
			},
			expectedTimestamp: timestamp1,
		},
		{
			getMetricDataResults: []cloudwatch.MetricDataResult{
				{
					Id:         &id1,
					Label:      &label1,
					StatusCode: cloudwatch.StatusCodeComplete,
					Timestamps: []time.Time{timestamp1, timestamp2},
					Values:     []float64{0, 1},
				},
				{
					Id:         &id2,
					Label:      &label2,
					StatusCode: cloudwatch.StatusCodeComplete,
				},
			},
			expectedTimestamp: timestamp1,
		},
		{
			getMetricDataResults: []cloudwatch.MetricDataResult{
				{
					Id:         &id1,
					Label:      &label1,
					StatusCode: cloudwatch.StatusCodeComplete,
					Timestamps: []time.Time{timestamp1, timestamp2},
					Values:     []float64{0, 1},
				},
				{
					Id:         &id2,
					Label:      &label2,
					StatusCode: cloudwatch.StatusCodeComplete,
				},
				{
					Id:         &id3,
					Label:      &label2,
					StatusCode: cloudwatch.StatusCodeComplete,
					Timestamps: []time.Time{timestamp2},
					Values:     []float64{2, 3},
				},
			},
			expectedTimestamp: timestamp2,
		},
	}

	for _, c := range cases {
		outputTimestamp := FindTimestamp(c.getMetricDataResults)
		assert.Equal(t, c.expectedTimestamp, outputTimestamp)
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

	expectedResourceTagMap := map[string][]resourcegroupstaggingapi.Tag{}
	expectedResourceTagMap["mysql-db-1"] = []resourcegroupstaggingapi.Tag{
		{
			Key:   awssdk.String("organization"),
			Value: awssdk.String("engineering"),
		},
		{
			Key:   awssdk.String("owner"),
			Value: awssdk.String("foo"),
		},
	}
	expectedResourceTagMap["mysql-db-2"] = []resourcegroupstaggingapi.Tag{
		{
			Key:   awssdk.String("organization"),
			Value: awssdk.String("finance"),
		},
		{
			Key:   awssdk.String("owner"),
			Value: awssdk.String("boo"),
		},
	}
	expectedResourceTagMap["db:mysql-db-1"] = []resourcegroupstaggingapi.Tag{
		{
			Key:   awssdk.String("organization"),
			Value: awssdk.String("engineering"),
		},
		{
			Key:   awssdk.String("owner"),
			Value: awssdk.String("foo"),
		},
	}
	expectedResourceTagMap["db:mysql-db-2"] = []resourcegroupstaggingapi.Tag{
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
