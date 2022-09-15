// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package cloudwatchsynthetics

import (
	"context"
	"errors"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/awsV1"
)

var (
	value1      = 0.25
	value2      = 5.0
	regionName  = "us-west-1"
	accountID   = "123456789012"
	accountName = "test"
	timestamp   = time.Date(2020, 10, 06, 00, 00, 00, 0, time.UTC)
	id1         = "cpu"
	label1      = "CPUUtilization|AWS/EC2|Average|InstanceId|i-1"
	id2         = "disk"
	label2      = "DiskReadOps|AWS/EC2|Average|InstanceId|i-1"
	label3      = "CPUUtilization|AWS/EC2|Average"
	label4      = "DiskReadOps|AWS/EC2|Average"
	instanceID1 = "i-1"
	instanceID2 = "i-2"
	namespace   = "AWS/EC2"
	dimName     = "InstanceId"
	metricName1 = "CPUUtilization"
	metricName2 = "StatusCheckFailed"
	metricName3 = "StatusCheckFailed_System"
	metricName4 = "StatusCheckFailed_Instance"
	listMetric1 = cloudwatch.Metric{
		Dimensions: awsV1.PointersOf([]cloudwatch.Dimension{{
			Name:  &dimName,
			Value: &instanceID1,
		}}).([]*cloudwatch.Dimension),
		MetricName: &metricName1,
		Namespace:  &namespace,
	}

	listMetric2 = cloudwatch.Metric{
		Dimensions: awsV1.PointersOf([]cloudwatch.Dimension{{
			Name:  &dimName,
			Value: &instanceID1,
		}}).([]*cloudwatch.Dimension),
		MetricName: &metricName2,
		Namespace:  &namespace,
	}

	listMetric3 = cloudwatch.Metric{
		Dimensions: awsV1.PointersOf([]cloudwatch.Dimension{{
			Name:  &dimName,
			Value: &instanceID2,
		}}).([]*cloudwatch.Dimension),
		MetricName: &metricName3,
		Namespace:  &namespace,
	}

	listMetric4 = cloudwatch.Metric{
		Dimensions: awsV1.PointersOf([]cloudwatch.Dimension{{
			Name:  &dimName,
			Value: &instanceID2,
		}}).([]*cloudwatch.Dimension),
		MetricName: &metricName4,
		Namespace:  &namespace,
	}

	listMetric5 = cloudwatch.Metric{
		MetricName: &metricName1,
		Namespace:  &namespace,
	}

	namespaceMSK = "AWS/Kafka"
	metricName6  = "MemoryUsed"
	listMetric8  = cloudwatch.Metric{
		MetricName: &metricName6,
		Namespace:  &namespaceMSK,
	}
)

type MockResourceGroupsTaggingClient struct{}

// GetResources implements resourcegroupstaggingapi.GetResourcesAPIClient.
func (m *MockResourceGroupsTaggingClient) GetResources(context.Context, *resourcegroupstaggingapi.GetResourcesInput) (*resourcegroupstaggingapi.GetResourcesOutput, error) {
	return &resourcegroupstaggingapi.GetResourcesOutput{
		PaginationToken: awssdk.String(""),
		ResourceTagMappingList: awsV1.PointersOf([]resourcegroupstaggingapi.ResourceTagMapping{
			{
				ResourceARN: awssdk.String("arn:aws:ec2:us-west-1:123456789012:instance:i-1"),
				Tags: awsV1.PointersOf([]resourcegroupstaggingapi.Tag{
					{
						Key:   awssdk.String("name"),
						Value: awssdk.String("test-ec2"),
					},
				}).([]*resourcegroupstaggingapi.Tag),
			},
		}).([]*resourcegroupstaggingapi.ResourceTagMapping),
	}, nil
}

type MockCloudWatchClient struct{}

// GetMetricData implements cloudwatch.GetMetricDataAPIClient interface
func (m *MockCloudWatchClient) GetMetricData(context.Context, *cloudwatch.GetMetricDataInput) (*cloudwatch.GetMetricDataOutput, error) {
	emptyString := ""
	return &cloudwatch.GetMetricDataOutput{
		Messages: nil,
		MetricDataResults: awsV1.PointersOf([]cloudwatch.MetricDataResult{
			{
				Id:         &id1,
				Label:      &label1,
				Values:     []*float64{&value1},
				Timestamps: []*time.Time{&timestamp},
			},
			{
				Id:         &id2,
				Label:      &label2,
				Values:     []*float64{&value2},
				Timestamps: []*time.Time{&timestamp},
			},
		}).([]*cloudwatch.MetricDataResult),
		NextToken: &emptyString,
	}, nil
}

type MockCloudWatchClientWithoutDim struct{}

// GetMetricData implements cloudwatch.GetMetricDataAPIClient.
func (m *MockCloudWatchClientWithoutDim) GetMetricData(context.Context, *cloudwatch.GetMetricDataInput) (*cloudwatch.GetMetricDataOutput, error) {
	emptyString := ""
	return &cloudwatch.GetMetricDataOutput{
		Messages: nil,
		MetricDataResults: awsV1.PointersOf([]cloudwatch.MetricDataResult{
			{
				Id:         &id1,
				Label:      &label3,
				Values:     []*float64{&value1},
				Timestamps: []*time.Time{&timestamp},
			},
			{
				Id:         &id2,
				Label:      &label4,
				Values:     []*float64{&value2},
				Timestamps: []*time.Time{&timestamp},
			},
		}).([]*cloudwatch.MetricDataResult),
		NextToken: &emptyString,
	}, nil
}

func TestConstructLabel(t *testing.T) {
	cases := []struct {
		listMetricDetail cloudwatch.Metric
		statistic        string
		expectedLabel    string
	}{
		{
			listMetric1,
			"Average",
			"CPUUtilization|AWS/EC2|Average|InstanceId|i-1",
		},
		{
			listMetric2,
			"Maximum",
			"StatusCheckFailed|AWS/EC2|Maximum|InstanceId|i-1",
		},
		{
			listMetric3,
			"Minimum",
			"StatusCheckFailed_System|AWS/EC2|Minimum|InstanceId|i-2",
		},
		{
			listMetric4,
			"Sum",
			"StatusCheckFailed_Instance|AWS/EC2|Sum|InstanceId|i-2",
		},
		{
			listMetric5,
			"SampleCount",
			"CPUUtilization|AWS/EC2|SampleCount",
		},
		{
			listMetric8,
			"SampleCount",
			"MemoryUsed|AWS/Kafka|SampleCount",
		},
	}

	for _, c := range cases {
		label := constructLabel(c.listMetricDetail, c.statistic)
		assert.Equal(t, c.expectedLabel, label)
	}
}

func TestCompareAWSDimensions(t *testing.T) {
	cases := []struct {
		title          string
		dim1           []cloudwatch.Dimension
		dim2           []cloudwatch.Dimension
		expectedResult bool
	}{
		{
			"same dimensions with length 2 but different order",
			[]cloudwatch.Dimension{
				{Name: awssdk.String("dept"), Value: awssdk.String("engineering")},
				{Name: awssdk.String("owner"), Value: awssdk.String("ks")},
			},
			[]cloudwatch.Dimension{
				{Name: awssdk.String("owner"), Value: awssdk.String("ks")},
				{Name: awssdk.String("dept"), Value: awssdk.String("engineering")},
			},
			true,
		},
		{
			"different dimensions with different length",
			[]cloudwatch.Dimension{
				{Name: awssdk.String("dept"), Value: awssdk.String("engineering")},
				{Name: awssdk.String("owner"), Value: awssdk.String("ks")},
			},
			[]cloudwatch.Dimension{
				{Name: awssdk.String("dept"), Value: awssdk.String("engineering")},
			},
			false,
		},
		{
			"different dimensions with same length",
			[]cloudwatch.Dimension{
				{Name: awssdk.String("owner"), Value: awssdk.String("ks")},
			},
			[]cloudwatch.Dimension{
				{Name: awssdk.String("dept"), Value: awssdk.String("engineering")},
			},
			false,
		},
		{
			"compare with an empty dimension",
			[]cloudwatch.Dimension{
				{Name: awssdk.String("owner"), Value: awssdk.String("ks")},
			},
			[]cloudwatch.Dimension{},
			false,
		},
		{
			"compare with wildcard dimension value, one same name dimension",
			[]cloudwatch.Dimension{
				{Name: awssdk.String("ID1"), Value: awssdk.String("111")},
			},
			[]cloudwatch.Dimension{
				{Name: awssdk.String("ID1"), Value: awssdk.String(dimensionValueWildcard)},
			},
			true,
		},
		{
			"compare with wildcard dimension value, one different name dimension",
			[]cloudwatch.Dimension{
				{Name: awssdk.String("IDx"), Value: awssdk.String("111")},
			},
			[]cloudwatch.Dimension{
				{Name: awssdk.String("ID1"), Value: awssdk.String(dimensionValueWildcard)},
			},
			false,
		},
		{
			"compare with wildcard dimension value, two same name dimensions",
			[]cloudwatch.Dimension{
				{Name: awssdk.String("ID1"), Value: awssdk.String("111")},
				{Name: awssdk.String("ID2"), Value: awssdk.String("222")},
			},
			[]cloudwatch.Dimension{
				{Name: awssdk.String("ID1"), Value: awssdk.String("111")},
				{Name: awssdk.String("ID2"), Value: awssdk.String(dimensionValueWildcard)},
			},
			true,
		},
		{
			"compare with wildcard dimension value, different length, case1",
			[]cloudwatch.Dimension{
				{Name: awssdk.String("ID1"), Value: awssdk.String("111")},
				{Name: awssdk.String("ID2"), Value: awssdk.String("222")},
			},
			[]cloudwatch.Dimension{
				{Name: awssdk.String("ID2"), Value: awssdk.String(dimensionValueWildcard)},
			},
			false,
		},
		{
			"compare with wildcard dimension value, different length, case2",
			[]cloudwatch.Dimension{
				{Name: awssdk.String("ID1"), Value: awssdk.String("111")},
			},
			[]cloudwatch.Dimension{
				{Name: awssdk.String("ID1"), Value: awssdk.String("111")},
				{Name: awssdk.String("ID2"), Value: awssdk.String(dimensionValueWildcard)},
			},
			false,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			output := compareAWSDimensions(c.dim1, c.dim2)
			assert.Equal(t, c.expectedResult, output)
		})
	}
}

func TestConstructTagsFilters(t *testing.T) {
	expectedResourceTypeTagFiltersEC2 := map[string][]aws.Tag{}
	expectedResourceTypeTagFiltersEC2["ec2:instance"] = nil

	expectedResourceTypeTagFiltersELB := map[string][]aws.Tag{}
	expectedResourceTypeTagFiltersELB["elasticloadbalancing"] = []aws.Tag{
		{
			Key:   "name",
			Value: []string{"test-elb1"},
		},
		{
			Key:   "name",
			Value: []string{"test-elb2"},
		},
	}

	expectedResourceTypeTagFiltersELBEC2 := map[string][]aws.Tag{}
	expectedResourceTypeTagFiltersELBEC2["elasticloadbalancing"] = []aws.Tag{
		{
			Key:   "name",
			Value: []string{"test-elb"},
		},
	}
	expectedResourceTypeTagFiltersELBEC2["ec2:instance"] = []aws.Tag{
		{
			Key:   "name",
			Value: []string{"test-ec2"},
		},
	}

	cases := []struct {
		title                  string
		namespaceDetails       []namespaceDetail
		resourceTypeTagFilters map[string][]aws.Tag
	}{
		{
			"test with one config per namespace",
			[]namespaceDetail{
				{
					resourceTypeFilter: "ec2:instance",
					statistics:         []string{"Average"},
					dimensions: []cloudwatch.Dimension{
						{
							Name:  awssdk.String("InstanceId"),
							Value: awssdk.String("i-1"),
						},
					},
				},
			},
			expectedResourceTypeTagFiltersEC2,
		},
		{
			"test with two configs for the same namespace",
			[]namespaceDetail{
				{
					resourceTypeFilter: "elasticloadbalancing",
					names:              []string{"BackendConnectionErrors", "HTTPCode_Backend_2XX", "HTTPCode_Backend_3XX"},
					statistics:         []string{"Sum"},
					tags: []aws.Tag{
						{
							Key:   "name",
							Value: []string{"test-elb1"},
						},
					},
				},
				{
					resourceTypeFilter: "elasticloadbalancing",
					names:              []string{"HealthyHostCount", "SurgeQueueLength", "UnHealthyHostCount"},
					statistics:         []string{"Maximum"},
					tags: []aws.Tag{
						{
							Key:   "name",
							Value: []string{"test-elb2"},
						},
					},
				},
			},
			expectedResourceTypeTagFiltersELB,
		},
		{
			"test with two configs for different namespaces",
			[]namespaceDetail{
				{
					resourceTypeFilter: "elasticloadbalancing",
					names:              []string{"BackendConnectionErrors", "HTTPCode_Backend_2XX", "HTTPCode_Backend_3XX"},
					statistics:         []string{"Sum"},
					tags: []aws.Tag{
						{
							Key:   "name",
							Value: []string{"test-elb"},
						},
					},
				},
				{
					resourceTypeFilter: "ec2:instance",
					names:              []string{"CPUUtilization"},
					statistics:         defaultStatistics,
					tags: []aws.Tag{
						{
							Key:   "name",
							Value: []string{"test-ec2"},
						},
					},
				},
			},
			expectedResourceTypeTagFiltersELBEC2,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			output := constructTagsFilters(c.namespaceDetails)
			assert.Equal(t, c.resourceTypeTagFilters, output)
		})
	}
}

func TestFilterListMetricsOutput(t *testing.T) {
	cases := []struct {
		title                        string
		listMetricsOutput            []cloudwatch.Metric
		namespaceDetails             []namespaceDetail
		filteredMetricWithStatsTotal []metricsWithStatistics
	}{
		{
			"test filter cloudwatch metrics with dimension",
			[]cloudwatch.Metric{
				{
					Dimensions: awsV1.PointersOf([]cloudwatch.Dimension{
						{
							Name:  awssdk.String("DBClusterIdentifier"),
							Value: awssdk.String("test1-cluster"),
						},
						{
							Name:  awssdk.String("Role"),
							Value: awssdk.String("READER"),
						}}).([]*cloudwatch.Dimension),
					MetricName: awssdk.String("CommitThroughput"),
					Namespace:  awssdk.String("AWS/RDS"),
				},
				{
					Dimensions: awsV1.PointersOf([]cloudwatch.Dimension{{
						Name:  awssdk.String("InstanceId"),
						Value: awssdk.String("i-1"),
					}}).([]*cloudwatch.Dimension),
					MetricName: awssdk.String("CPUUtilization"),
					Namespace:  awssdk.String("AWS/EC2"),
				},
			},
			[]namespaceDetail{
				{
					resourceTypeFilter: "ec2:instance",
					statistics:         []string{"Average"},
					dimensions: []cloudwatch.Dimension{
						{
							Name:  awssdk.String("InstanceId"),
							Value: awssdk.String("i-1"),
						},
					},
				},
			},
			[]metricsWithStatistics{
				{
					cloudwatch.Metric{
						Dimensions: awsV1.PointersOf([]cloudwatch.Dimension{{
							Name:  awssdk.String("InstanceId"),
							Value: awssdk.String("i-1"),
						}}).([]*cloudwatch.Dimension),
						MetricName: awssdk.String("CPUUtilization"),
						Namespace:  awssdk.String("AWS/EC2"),
					},
					[]string{"Average"},
				},
			},
		},
		{
			"test filter cloudwatch metrics with name",
			[]cloudwatch.Metric{
				{
					Dimensions: awsV1.PointersOf([]cloudwatch.Dimension{
						{
							Name:  awssdk.String("DBClusterIdentifier"),
							Value: awssdk.String("test1-cluster"),
						},
						{
							Name:  awssdk.String("Role"),
							Value: awssdk.String("READER"),
						}}).([]*cloudwatch.Dimension),
					MetricName: awssdk.String("CommitThroughput"),
					Namespace:  awssdk.String("AWS/RDS"),
				},
				{
					Dimensions: awsV1.PointersOf([]cloudwatch.Dimension{{
						Name:  awssdk.String("InstanceId"),
						Value: awssdk.String("i-1"),
					}}).([]*cloudwatch.Dimension),
					MetricName: awssdk.String("CPUUtilization"),
					Namespace:  awssdk.String("AWS/EC2"),
				},
			},
			[]namespaceDetail{
				{
					names:              []string{"CPUUtilization"},
					resourceTypeFilter: "ec2:instance",
					statistics:         []string{"Average"},
				},
			},
			[]metricsWithStatistics{
				{
					cloudwatch.Metric{
						Dimensions: awsV1.PointersOf([]cloudwatch.Dimension{{
							Name:  awssdk.String("InstanceId"),
							Value: awssdk.String("i-1"),
						}}).([]*cloudwatch.Dimension),
						MetricName: awssdk.String("CPUUtilization"),
						Namespace:  awssdk.String("AWS/EC2"),
					},
					[]string{"Average"},
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			output := filterListMetricsOutput(c.listMetricsOutput, c.namespaceDetails)
			assert.Equal(t, c.filteredMetricWithStatsTotal, output)
		})
	}
}

func TestCheckStatistics(t *testing.T) {
	m := MetricSet{}
	cases := []struct {
		title           string
		statisticMethod string
		expectedOutput  error
	}{
		{
			"test average",
			"Average",
			nil,
		},
		{
			"test sum",
			"Sum",
			nil,
		},
		{
			"test max",
			"Maximum",
			nil,
		},
		{
			"test min",
			"Minimum",
			nil,
		},
		{
			"test count",
			"SampleCount",
			nil,
		},
		{
			"test pN",
			"p10",
			nil,
		},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			m.CloudwatchConfigs = []Config{{Statistic: []string{c.statisticMethod}}}
			output := m.checkStatistics()
			assert.Equal(t, c.expectedOutput, output)
		})
	}

	casesFailed := []struct {
		title            string
		statisticMethods []string
		expectedOutput   error
	}{
		{
			"wrong statistic method",
			[]string{"test"},
			errors.New("statistic method specified is not valid: test"),
		},
		{
			"one correct and one wrong statistic method",
			[]string{"Sum", "test"},
			errors.New("statistic method specified is not valid: test"),
		},
	}
	for _, c := range casesFailed {
		t.Run(c.title, func(t *testing.T) {
			m.CloudwatchConfigs = []Config{{Statistic: c.statisticMethods}}
			output := m.checkStatistics()
			assert.Error(t, output)
		})
	}
}

func TestInsertTags(t *testing.T) {
	identifier1 := "StandardStorage,test-s3-1"
	identifier2 := "test-s3-2"
	tagKey1 := "organization"
	tagValue1 := "engineering"
	tagKey2 := "owner"
	tagValue2 := "foo"
	identifierContainsArn := "arn:aws:ec2:ap-northeast-1:111111111111:eip-allocation/eipalloc-0123456789abcdef,SYNFlood"
	tagKey3 := "env"
	tagValue3 := "dev"

	events := map[string]mb.Event{}
	events[identifier1] = aws.InitEvent(regionName, accountName, accountID, timestamp)
	events[identifier2] = aws.InitEvent(regionName, accountName, accountID, timestamp)
	events[identifierContainsArn] = aws.InitEvent(regionName, accountName, accountID, timestamp)

	resourceTagMap := map[string][]resourcegroupstaggingapi.Tag{}
	resourceTagMap["test-s3-1"] = []resourcegroupstaggingapi.Tag{
		{
			Key:   awssdk.String(tagKey1),
			Value: awssdk.String(tagValue1),
		},
	}
	resourceTagMap["test-s3-2"] = []resourcegroupstaggingapi.Tag{
		{
			Key:   awssdk.String(tagKey2),
			Value: awssdk.String(tagValue2),
		},
	}
	resourceTagMap["eipalloc-0123456789abcdef"] = []resourcegroupstaggingapi.Tag{
		{
			Key:   awssdk.String(tagKey3),
			Value: awssdk.String(tagValue3),
		},
	}

	cases := []struct {
		title            string
		identifier       string
		expectedTagKey   string
		expectedTagValue string
	}{
		{
			"test identifier with storage type and s3 bucket name",
			identifier1,
			"aws.tags.organization",
			tagValue1,
		},
		{
			"test identifier with only s3 bucket name",
			identifier2,
			"aws.tags.owner",
			tagValue2,
		},
		{
			"test identifier with arn value",
			identifierContainsArn,
			"aws.tags.env",
			tagValue3,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			insertTags(events, c.identifier, resourceTagMap)
			value, err := events[c.identifier].RootFields.GetValue(c.expectedTagKey)
			assert.NoError(t, err)
			assert.Equal(t, c.expectedTagValue, value)
		})
	}
}

func TestConfigDimensionValueContainsWildcard(t *testing.T) {
	instanceId := "InstanceId"
	i11 := "i-111111"
	i22 := "i-2222"
	cases := []struct {
		title          string
		dimensions     []cloudwatch.Dimension
		expectedResult bool
	}{
		{
			"test dimensions without wolidcard value",
			[]cloudwatch.Dimension{
				{
					Name:  &instanceId,
					Value: &i11,
				},
				{
					Name:  &instanceId,
					Value: &i22,
				},
			},
			false,
		},
		{
			"test dimensions without wolidcard value",
			[]cloudwatch.Dimension{
				{
					Name:  &instanceId,
					Value: &i11,
				},
				{
					Name:  &instanceId,
					Value: &dimensionValueWildcard,
				},
			},
			true,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			result := configDimensionValueContainsWildcard(c.dimensions)
			assert.Equal(t, c.expectedResult, result)
		})
	}
}
