// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package cloudwatch

import (
	"context"
	"errors"
	cloudwatchtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	resourcegroupstaggingapitypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
	"github.com/aws/smithy-go/middleware"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
)

var (
	regionName  = "us-west-1"
	timestamp   = time.Date(2020, 10, 06, 00, 00, 00, 0, time.UTC)
	accountID   = "123456789012"
	accountName = "test"

	id1    = "cpu"
	value1 = 0.25
	label1 = "CPUUtilization|AWS/EC2|Average|InstanceId|i-1"

	id2    = "disk"
	value2 = 5.0
	label2 = "DiskReadOps|AWS/EC2|Average|InstanceId|i-1"

	label3 = "CPUUtilization|AWS/EC2|Average"
	label4 = "DiskReadOps|AWS/EC2|Average"

	instanceID1     = "i-1"
	instanceID2     = "i-2"
	namespace       = "AWS/EC2"
	dimName         = "InstanceId"
	metricName1     = "CPUUtilization"
	metricName2     = "StatusCheckFailed"
	metricName3     = "StatusCheckFailed_System"
	metricName4     = "StatusCheckFailed_Instance"
	resourceTypeEC2 = "ec2:instance"
	listMetric1     = cloudwatchtypes.Metric{
		Dimensions: []cloudwatchtypes.Dimension{{
			Name:  &dimName,
			Value: &instanceID1,
		}},
		MetricName: &metricName1,
		Namespace:  &namespace,
	}

	listMetric2 = cloudwatchtypes.Metric{
		Dimensions: []cloudwatchtypes.Dimension{{
			Name:  &dimName,
			Value: &instanceID1,
		}},
		MetricName: &metricName2,
		Namespace:  &namespace,
	}

	listMetric3 = cloudwatchtypes.Metric{
		Dimensions: []cloudwatchtypes.Dimension{{
			Name:  &dimName,
			Value: &instanceID2,
		}},
		MetricName: &metricName3,
		Namespace:  &namespace,
	}

	listMetric4 = cloudwatchtypes.Metric{
		Dimensions: []cloudwatchtypes.Dimension{{
			Name:  &dimName,
			Value: &instanceID2,
		}},
		MetricName: &metricName4,
		Namespace:  &namespace,
	}

	listMetric5 = cloudwatchtypes.Metric{
		MetricName: &metricName1,
		Namespace:  &namespace,
	}

	namespaceMSK = "AWS/Kafka"
	metricName6  = "MemoryUsed"
	listMetric8  = cloudwatchtypes.Metric{
		MetricName: &metricName6,
		Namespace:  &namespaceMSK,
	}
)

func TestConstructLabel(t *testing.T) {
	cases := []struct {
		listMetricDetail cloudwatchtypes.Metric
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

func TestReadCloudwatchConfig(t *testing.T) {
	m := MetricSet{}
	m.MetricSet = &aws.MetricSet{Period: 5}
	resourceTypeFiltersEC2 := map[string][]aws.Tag{}
	resourceTypeFiltersEC2["ec2:instance"] = nil

	expectedListMetricWithDetailEC2 := listMetricWithDetail{
		metricsWithStats: []metricsWithStatistics{
			{
				cloudwatchtypes.Metric{
					Dimensions: []cloudwatchtypes.Dimension{{
						Name:  awssdk.String("InstanceId"),
						Value: awssdk.String("i-1"),
					}},
					MetricName: awssdk.String("CPUUtilization"),
					Namespace:  awssdk.String("AWS/EC2"),
				},
				[]string{"Average"},
				nil,
			},
		},
		resourceTypeFilters: resourceTypeFiltersEC2,
	}

	expectedNamespaceWithDetailS3 := map[string][]namespaceDetail{}
	expectedNamespaceWithDetailS3["AWS/S3"] = []namespaceDetail{
		{
			statistics: defaultStatistics,
		},
	}

	resourceTypeFiltersEC2RDS := map[string][]aws.Tag{}
	resourceTypeFiltersEC2RDS["ec2:instance"] = nil
	resourceTypeFiltersEC2RDS["rds"] = nil

	expectedListMetricWithDetailEC2RDS := listMetricWithDetail{
		metricsWithStats: []metricsWithStatistics{
			{
				cloudwatchtypes.Metric{
					Dimensions: []cloudwatchtypes.Dimension{{
						Name:  awssdk.String("InstanceId"),
						Value: awssdk.String("i-1"),
					}},
					MetricName: awssdk.String("CPUUtilization"),
					Namespace:  awssdk.String("AWS/EC2"),
				},
				[]string{"Average"},
				nil,
			},
			{
				cloudwatchtypes.Metric{
					Dimensions: []cloudwatchtypes.Dimension{{
						Name:  awssdk.String("DBClusterIdentifier"),
						Value: awssdk.String("test1-cluster"),
					},
						{
							Name:  awssdk.String("Role"),
							Value: awssdk.String("READER"),
						}},
					MetricName: awssdk.String("CommitThroughput"),
					Namespace:  awssdk.String("AWS/RDS"),
				},
				[]string{"Average"},
				nil,
			},
		},
		resourceTypeFilters: resourceTypeFiltersEC2RDS,
	}

	resourceTypeFiltersEC2RDSWithTag := map[string][]aws.Tag{}
	resourceTypeFiltersEC2RDSWithTag["ec2:instance"] = []aws.Tag{
		{
			Key:   "name",
			Value: "test",
		},
	}
	resourceTypeFiltersEC2RDSWithTag["rds"] = []aws.Tag{
		{
			Key:   "name",
			Value: "test",
		},
	}
	expectedListMetricWithDetailEC2RDSWithTag := listMetricWithDetail{
		metricsWithStats: []metricsWithStatistics{
			{
				cloudwatchtypes.Metric{
					Dimensions: []cloudwatchtypes.Dimension{{
						Name:  awssdk.String("InstanceId"),
						Value: awssdk.String("i-1"),
					}},
					MetricName: awssdk.String("CPUUtilization"),
					Namespace:  awssdk.String("AWS/EC2"),
				},
				[]string{"Average"},
				nil,
			},
			{
				cloudwatchtypes.Metric{
					Dimensions: []cloudwatchtypes.Dimension{{
						Name:  awssdk.String("DBClusterIdentifier"),
						Value: awssdk.String("test1-cluster"),
					},
						{
							Name:  awssdk.String("Role"),
							Value: awssdk.String("READER"),
						}},
					MetricName: awssdk.String("CommitThroughput"),
					Namespace:  awssdk.String("AWS/RDS"),
				},
				[]string{"Average"},
				nil,
			},
		},
		resourceTypeFilters: resourceTypeFiltersEC2RDSWithTag,
	}

	expectedNamespaceDetailLambda := map[string][]namespaceDetail{}
	expectedNamespaceDetailLambda["AWS/Lambda"] = []namespaceDetail{
		{
			statistics: defaultStatistics,
		},
	}

	expectedNamespaceWithDetailEC2S3 := map[string][]namespaceDetail{}
	expectedNamespaceWithDetailEC2S3["AWS/EC2"] = []namespaceDetail{
		{
			resourceTypeFilter: "ec2:instance",
			names:              []string{"CPUUtilization"},
			statistics:         defaultStatistics,
		},
	}
	expectedNamespaceWithDetailEC2S3["AWS/S3"] = []namespaceDetail{
		{
			resourceTypeFilter: "s3",
			statistics:         defaultStatistics,
		},
	}

	expectedNamespaceWithDetailEBS := map[string][]namespaceDetail{}
	expectedNamespaceWithDetailEBS["AWS/EBS"] = []namespaceDetail{
		{
			resourceTypeFilter: "ec2",
			statistics:         defaultStatistics,
		},
	}

	expectedNamespaceWithDetailEC2 := map[string][]namespaceDetail{}
	expectedNamespaceWithDetailEC2["AWS/EC2"] = []namespaceDetail{
		{
			resourceTypeFilter: "ec2:instance",
			names:              []string{"CPUUtilization", "StatusCheckFailed"},
			statistics:         []string{"Average", "Maximum"},
		},
	}

	expectedNamespaceWithDetailKafka := map[string][]namespaceDetail{}
	expectedNamespaceWithDetailKafka["AWS/Kafka"] = []namespaceDetail{
		{
			names:      []string{"MemoryUsed"},
			statistics: defaultStatistics,
		},
	}

	expectedNamespaceDetailTotal := map[string][]namespaceDetail{}
	expectedNamespaceDetailTotal["AWS/EC2"] = []namespaceDetail{
		{
			resourceTypeFilter: resourceTypeEC2,
			names:              []string{"CPUUtilization"},
			statistics:         defaultStatistics,
			tags: []aws.Tag{
				{
					Key:   "name",
					Value: "test",
				},
			},
		},
	}
	expectedNamespaceDetailTotal["AWS/ELB"] = []namespaceDetail{
		{
			resourceTypeFilter: "elasticloadbalancing",
			names:              []string{"BackendConnectionErrors", "HTTPCode_Backend_2XX", "HTTPCode_Backend_3XX"},
			statistics:         []string{"Sum"},
			tags: []aws.Tag{
				{
					Key:   "name",
					Value: "test",
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
					Value: "test",
				},
			},
		},
	}

	expectedNamespaceDetailELBLambda := map[string][]namespaceDetail{}
	expectedNamespaceDetailELBLambda["AWS/Lambda"] = []namespaceDetail{
		{
			statistics: defaultStatistics,
			tags: []aws.Tag{
				{
					Key:   "name",
					Value: "test",
				},
			},
		},
	}
	expectedNamespaceDetailELBLambda["AWS/ELB"] = []namespaceDetail{
		{
			resourceTypeFilter: "elasticloadbalancing",
			names:              []string{"BackendConnectionErrors", "HTTPCode_Backend_2XX", "HTTPCode_Backend_3XX"},
			statistics:         []string{"Sum"},
			tags: []aws.Tag{
				{
					Key:   "name",
					Value: "test",
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
					Value: "test",
				},
			},
		},
	}

	expectedNamespaceWithDetailEC2WithNoMetricName := map[string][]namespaceDetail{}
	expectedNamespaceWithDetailEC2WithNoMetricName["AWS/EC2"] = []namespaceDetail{
		{
			resourceTypeFilter: "ec2:instance",
			statistics:         []string{"Average"},
			dimensions: []cloudwatchtypes.Dimension{
				{
					Name:  awssdk.String("InstanceId"),
					Value: awssdk.String("i-1"),
				},
			},
		},
	}

	expectedListMetricsEC2WithDim := listMetricWithDetail{
		metricsWithStats: []metricsWithStatistics{
			{
				cloudwatchtypes.Metric{
					Dimensions: []cloudwatchtypes.Dimension{{
						Name:  awssdk.String("InstanceId"),
						Value: awssdk.String("i-1"),
					}},
					MetricName: awssdk.String("CPUUtilization"),
					Namespace:  awssdk.String("AWS/EC2"),
				},
				[]string{"Average"},
				nil,
			},
			{
				cloudwatchtypes.Metric{
					Dimensions: []cloudwatchtypes.Dimension{{
						Name:  awssdk.String("InstanceId"),
						Value: awssdk.String("i-1"),
					}},
					MetricName: awssdk.String("DiskReadOps"),
					Namespace:  awssdk.String("AWS/EC2"),
				},
				[]string{"Average"},
				nil,
			},
		},
		resourceTypeFilters: resourceTypeFiltersEC2,
	}

	cases := []struct {
		title                         string
		cloudwatchMetricsConfig       []Config
		tagsFilter                    []aws.Tag
		expectedListMetricDetailTotal listMetricWithDetail
		expectedNamespaceDetailTotal  map[string][]namespaceDetail
	}{
		{
			"test with a specific metric",
			[]Config{
				{
					Namespace:  "AWS/EC2",
					MetricName: []string{"CPUUtilization"},
					Dimensions: []Dimension{
						{
							Name:  "InstanceId",
							Value: "i-1",
						},
					},
					ResourceType: "ec2:instance",
					Statistic:    []string{"Average"},
				},
			},
			nil,
			expectedListMetricWithDetailEC2,
			map[string][]namespaceDetail{},
		},
		{
			"test with a specific metric and a namespace",
			[]Config{
				{
					Namespace:  "AWS/EC2",
					MetricName: []string{"CPUUtilization"},
					Dimensions: []Dimension{
						{
							Name:  "InstanceId",
							Value: "i-1",
						},
					},
					ResourceType: "ec2:instance",
					Statistic:    []string{"Average"},
				},
				{
					Namespace: "AWS/S3",
				},
			},
			nil,
			expectedListMetricWithDetailEC2,
			expectedNamespaceWithDetailS3,
		},
		{
			"test with two specific metrics and a namespace",
			[]Config{
				{
					Namespace:  "AWS/EC2",
					MetricName: []string{"CPUUtilization"},
					Dimensions: []Dimension{
						{
							Name:  "InstanceId",
							Value: "i-1",
						},
					},
					Statistic:    []string{"Average"},
					ResourceType: "ec2:instance",
				},
				{
					Namespace: "AWS/Lambda",
				},
				{
					Namespace:  "AWS/RDS",
					MetricName: []string{"CommitThroughput"},
					Dimensions: []Dimension{
						{
							Name:  "DBClusterIdentifier",
							Value: "test1-cluster",
						},
						{
							Name:  "Role",
							Value: "READER",
						},
					},
					Statistic:    []string{"Average"},
					ResourceType: "rds",
				},
			},
			nil,
			expectedListMetricWithDetailEC2RDS,
			expectedNamespaceDetailLambda,
		},
		{
			"Test a specific metric (only with metric name) and a namespace",
			[]Config{
				{
					Namespace:    "AWS/EC2",
					MetricName:   []string{"CPUUtilization"},
					ResourceType: resourceTypeEC2,
				},
				{
					Namespace:    "AWS/S3",
					ResourceType: "s3",
				},
			},
			nil,
			listMetricWithDetail{
				resourceTypeFilters: map[string][]aws.Tag{},
			},
			expectedNamespaceWithDetailEC2S3,
		},
		{
			"test EBS namespace",
			[]Config{
				{
					Namespace:    "AWS/EBS",
					ResourceType: "ec2",
				},
			},
			nil,
			listMetricWithDetail{
				resourceTypeFilters: map[string][]aws.Tag{},
			},
			expectedNamespaceWithDetailEBS,
		},
		{
			"test with two metrics and no dimension",
			[]Config{
				{
					Namespace:    "AWS/EC2",
					MetricName:   []string{"CPUUtilization", "StatusCheckFailed"},
					ResourceType: resourceTypeEC2,
					Statistic:    []string{"Average", "Maximum"},
				},
			},
			nil,
			listMetricWithDetail{
				resourceTypeFilters: map[string][]aws.Tag{},
			},
			expectedNamespaceWithDetailEC2,
		},
		{
			"test AWS/Kafka MemoryUsed",
			[]Config{
				{
					Namespace:  "AWS/Kafka",
					MetricName: []string{"MemoryUsed"},
				},
			},
			nil,
			listMetricWithDetail{
				resourceTypeFilters: map[string][]aws.Tag{},
			},
			expectedNamespaceWithDetailKafka,
		},
		{
			"Test with different statistics",
			[]Config{
				{
					Namespace:    "AWS/EC2",
					MetricName:   []string{"CPUUtilization"},
					ResourceType: resourceTypeEC2,
				},
				{
					Namespace:    "AWS/ELB",
					MetricName:   []string{"BackendConnectionErrors", "HTTPCode_Backend_2XX", "HTTPCode_Backend_3XX"},
					Statistic:    []string{"Sum"},
					ResourceType: "elasticloadbalancing",
				},
				{
					Namespace:    "AWS/ELB",
					MetricName:   []string{"HealthyHostCount", "SurgeQueueLength", "UnHealthyHostCount"},
					Statistic:    []string{"Maximum"},
					ResourceType: "elasticloadbalancing",
				},
			},
			[]aws.Tag{
				{
					Key:   "name",
					Value: "test",
				},
			},
			listMetricWithDetail{
				resourceTypeFilters: map[string][]aws.Tag{},
			},
			expectedNamespaceDetailTotal,
		},
		{
			"Test with different statistics and a specific metric",
			[]Config{
				{
					Namespace:    "AWS/ELB",
					MetricName:   []string{"BackendConnectionErrors", "HTTPCode_Backend_2XX", "HTTPCode_Backend_3XX"},
					Statistic:    []string{"Sum"},
					ResourceType: "elasticloadbalancing",
				},
				{
					Namespace:    "AWS/ELB",
					MetricName:   []string{"HealthyHostCount", "SurgeQueueLength", "UnHealthyHostCount"},
					Statistic:    []string{"Maximum"},
					ResourceType: "elasticloadbalancing",
				},
				{
					Namespace: "AWS/Lambda",
				},
				{
					Namespace:  "AWS/EC2",
					MetricName: []string{"CPUUtilization"},
					Dimensions: []Dimension{
						{
							Name:  "InstanceId",
							Value: "i-1",
						},
					},
					Statistic:    []string{"Average"},
					ResourceType: "ec2:instance",
				},
				{
					Namespace:  "AWS/RDS",
					MetricName: []string{"CommitThroughput"},
					Dimensions: []Dimension{
						{
							Name:  "DBClusterIdentifier",
							Value: "test1-cluster",
						},
						{
							Name:  "Role",
							Value: "READER",
						},
					},
					Statistic:    []string{"Average"},
					ResourceType: "rds",
				},
			},
			[]aws.Tag{
				{
					Key:   "name",
					Value: "test",
				},
			},
			expectedListMetricWithDetailEC2RDSWithTag,
			expectedNamespaceDetailELBLambda,
		},
		{
			"Test with no metric name",
			[]Config{
				{
					Namespace: "AWS/EC2",
					Dimensions: []Dimension{
						{
							Name:  "InstanceId",
							Value: "i-1",
						},
					},
					Statistic:    []string{"Average"},
					ResourceType: "ec2:instance",
				},
			},
			nil,
			listMetricWithDetail{
				resourceTypeFilters: map[string][]aws.Tag{},
			},
			expectedNamespaceWithDetailEC2WithNoMetricName,
		},
		{
			"test with two metric names and a set of dimension",
			[]Config{
				{
					Namespace:  "AWS/EC2",
					MetricName: []string{"CPUUtilization", "DiskReadOps"},
					Dimensions: []Dimension{
						{
							Name:  "InstanceId",
							Value: "i-1",
						},
					},
					ResourceType: "ec2:instance",
					Statistic:    []string{"Average"},
				},
			},
			nil,
			expectedListMetricsEC2WithDim,
			map[string][]namespaceDetail{},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			m.CloudwatchConfigs = c.cloudwatchMetricsConfig
			m.MetricSet.TagsFilter = c.tagsFilter
			listMetricDetailTotal, namespaceDetailTotal := m.readCloudwatchConfig()
			assert.Equal(t, c.expectedListMetricDetailTotal, listMetricDetailTotal)
			assert.Equal(t, c.expectedNamespaceDetailTotal, namespaceDetailTotal)
		})
	}
}

func TestGenerateFieldName(t *testing.T) {
	cases := []struct {
		title             string
		metricsetName     string
		label             []string
		expectedFieldName string
	}{
		{
			"test Average",
			"cloudwatch",
			[]string{"CPUUtilization", "AWS/EC2", "Average", "InstanceId", "i-1"},
			"aws.ec2.metrics.CPUUtilization.avg",
		},
		{
			"test Maximum",
			"cloudwatch",
			[]string{"CPUUtilization", "AWS/EC2", "Maximum", "InstanceId", "i-1"},
			"aws.ec2.metrics.CPUUtilization.max",
		},
		{
			"test Minimum",
			"cloudwatch",
			[]string{"CPUUtilization", "AWS/EC2", "Minimum", "InstanceId", "i-1"},
			"aws.ec2.metrics.CPUUtilization.min",
		},
		{
			"test Sum",
			"cloudwatch",
			[]string{"CPUUtilization", "AWS/EC2", "Sum", "InstanceId", "i-1"},
			"aws.ec2.metrics.CPUUtilization.sum",
		},
		{
			"test SampleCount",
			"cloudwatch",
			[]string{"CPUUtilization", "AWS/EC2", "SampleCount", "InstanceId", "i-1"},
			"aws.ec2.metrics.CPUUtilization.count",
		},
		{
			"test extended statistic",
			"cloudwatch",
			[]string{"CPUUtilization", "AWS/EC2", "p10", "InstanceId", "i-1"},
			"aws.ec2.metrics.CPUUtilization.p10",
		},
		{
			"test other metricset",
			"ec2",
			[]string{"CPUUtilization", "AWS/EC2", "p10", "InstanceId", "i-1"},
			"aws.ec2.metrics.CPUUtilization.p10",
		},
		{
			"test metric name with dot",
			"cloudwatch",
			[]string{"DeliveryToS3.Records", "AWS/Firehose", "Average", "DeliveryStreamName", "test-1"},
			"aws.firehose.metrics.DeliveryToS3_Records.avg",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			fieldName := generateFieldName(c.label[namespaceIdx], c.label)
			assert.Equal(t, c.expectedFieldName, fieldName)
		})
	}
}

func TestCompareAWSDimensions(t *testing.T) {
	cases := []struct {
		title          string
		dim1           []cloudwatchtypes.Dimension
		dim2           []cloudwatchtypes.Dimension
		expectedResult bool
	}{
		{
			"same dimensions with length 2 but different order",
			[]cloudwatchtypes.Dimension{
				{Name: awssdk.String("dept"), Value: awssdk.String("engineering")},
				{Name: awssdk.String("owner"), Value: awssdk.String("ks")},
			},
			[]cloudwatchtypes.Dimension{
				{Name: awssdk.String("owner"), Value: awssdk.String("ks")},
				{Name: awssdk.String("dept"), Value: awssdk.String("engineering")},
			},
			true,
		},
		{
			"different dimensions with different length",
			[]cloudwatchtypes.Dimension{
				{Name: awssdk.String("dept"), Value: awssdk.String("engineering")},
				{Name: awssdk.String("owner"), Value: awssdk.String("ks")},
			},
			[]cloudwatchtypes.Dimension{
				{Name: awssdk.String("dept"), Value: awssdk.String("engineering")},
			},
			false,
		},
		{
			"different dimensions with same length",
			[]cloudwatchtypes.Dimension{
				{Name: awssdk.String("owner"), Value: awssdk.String("ks")},
			},
			[]cloudwatchtypes.Dimension{
				{Name: awssdk.String("dept"), Value: awssdk.String("engineering")},
			},
			false,
		},
		{
			"compare with an empty dimension",
			[]cloudwatchtypes.Dimension{
				{Name: awssdk.String("owner"), Value: awssdk.String("ks")},
			},
			[]cloudwatchtypes.Dimension{},
			false,
		},
		{
			"compare with wildcard dimension value, one same name dimension",
			[]cloudwatchtypes.Dimension{
				{Name: awssdk.String("ID1"), Value: awssdk.String("111")},
			},
			[]cloudwatchtypes.Dimension{
				{Name: awssdk.String("ID1"), Value: awssdk.String(dimensionValueWildcard)},
			},
			true,
		},
		{
			"compare with wildcard dimension value, one different name dimension",
			[]cloudwatchtypes.Dimension{
				{Name: awssdk.String("IDx"), Value: awssdk.String("111")},
			},
			[]cloudwatchtypes.Dimension{
				{Name: awssdk.String("ID1"), Value: awssdk.String(dimensionValueWildcard)},
			},
			false,
		},
		{
			"compare with wildcard dimension value, two same name dimensions",
			[]cloudwatchtypes.Dimension{
				{Name: awssdk.String("ID1"), Value: awssdk.String("111")},
				{Name: awssdk.String("ID2"), Value: awssdk.String("222")},
			},
			[]cloudwatchtypes.Dimension{
				{Name: awssdk.String("ID1"), Value: awssdk.String("111")},
				{Name: awssdk.String("ID2"), Value: awssdk.String(dimensionValueWildcard)},
			},
			true,
		},
		{
			"compare with wildcard dimension value, different length, case1",
			[]cloudwatchtypes.Dimension{
				{Name: awssdk.String("ID1"), Value: awssdk.String("111")},
				{Name: awssdk.String("ID2"), Value: awssdk.String("222")},
			},
			[]cloudwatchtypes.Dimension{
				{Name: awssdk.String("ID2"), Value: awssdk.String(dimensionValueWildcard)},
			},
			false,
		},
		{
			"compare with wildcard dimension value, different length, case2",
			[]cloudwatchtypes.Dimension{
				{Name: awssdk.String("ID1"), Value: awssdk.String("111")},
			},
			[]cloudwatchtypes.Dimension{
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
			Value: "test-elb1",
		},
		{
			Key:   "name",
			Value: "test-elb2",
		},
	}

	expectedResourceTypeTagFiltersELBEC2 := map[string][]aws.Tag{}
	expectedResourceTypeTagFiltersELBEC2["elasticloadbalancing"] = []aws.Tag{
		{
			Key:   "name",
			Value: "test-elb",
		},
	}
	expectedResourceTypeTagFiltersELBEC2["ec2:instance"] = []aws.Tag{
		{
			Key:   "name",
			Value: "test-ec2",
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
					dimensions: []cloudwatchtypes.Dimension{
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
							Value: "test-elb1",
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
							Value: "test-elb2",
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
							Value: "test-elb",
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
							Value: "test-ec2",
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
		listMetricsOutput            []cloudwatchtypes.Metric
		namespaceDetails             []namespaceDetail
		filteredMetricWithStatsTotal []metricsWithStatistics
	}{
		{
			"test filter cloudwatch metrics with dimension",
			[]cloudwatchtypes.Metric{
				{
					Dimensions: []cloudwatchtypes.Dimension{
						{
							Name:  awssdk.String("DBClusterIdentifier"),
							Value: awssdk.String("test1-cluster"),
						},
						{
							Name:  awssdk.String("Role"),
							Value: awssdk.String("READER"),
						}},
					MetricName: awssdk.String("CommitThroughput"),
					Namespace:  awssdk.String("AWS/RDS"),
				},
				{
					Dimensions: []cloudwatchtypes.Dimension{{
						Name:  awssdk.String("InstanceId"),
						Value: awssdk.String("i-1"),
					}},
					MetricName: awssdk.String("CPUUtilization"),
					Namespace:  awssdk.String("AWS/EC2"),
				},
			},
			[]namespaceDetail{
				{
					resourceTypeFilter: "ec2:instance",
					statistics:         []string{"Average"},
					dimensions: []cloudwatchtypes.Dimension{
						{
							Name:  awssdk.String("InstanceId"),
							Value: awssdk.String("i-1"),
						},
					},
				},
			},
			[]metricsWithStatistics{
				{
					cloudwatchtypes.Metric{
						Dimensions: []cloudwatchtypes.Dimension{{
							Name:  awssdk.String("InstanceId"),
							Value: awssdk.String("i-1"),
						}},
						MetricName: awssdk.String("CPUUtilization"),
						Namespace:  awssdk.String("AWS/EC2"),
					},
					[]string{"Average"},
					nil,
				},
			},
		},
		{
			"test filter cloudwatch metrics with name",
			[]cloudwatchtypes.Metric{
				{
					Dimensions: []cloudwatchtypes.Dimension{
						{
							Name:  awssdk.String("DBClusterIdentifier"),
							Value: awssdk.String("test1-cluster"),
						},
						{
							Name:  awssdk.String("Role"),
							Value: awssdk.String("READER"),
						}},
					MetricName: awssdk.String("CommitThroughput"),
					Namespace:  awssdk.String("AWS/RDS"),
				},
				{
					Dimensions: []cloudwatchtypes.Dimension{{
						Name:  awssdk.String("InstanceId"),
						Value: awssdk.String("i-1"),
					}},
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
					cloudwatchtypes.Metric{
						Dimensions: []cloudwatchtypes.Dimension{{
							Name:  awssdk.String("InstanceId"),
							Value: awssdk.String("i-1"),
						}},
						MetricName: awssdk.String("CPUUtilization"),
						Namespace:  awssdk.String("AWS/EC2"),
					},
					[]string{"Average"},
					nil,
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

// MockCloudWatchClient struct is used for unit tests.
type MockCloudWatchClient struct {
	//*cloudwatch.Client
}

// GetMetricData implements cloudwatch.GetMetricDataAPIClient interface
func (m *MockCloudWatchClient) GetMetricData(context.Context, *cloudwatch.GetMetricDataInput, ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
	emptyString := ""
	return &cloudwatch.GetMetricDataOutput{
		Messages: nil,
		MetricDataResults: []cloudwatchtypes.MetricDataResult{
			{
				Id:         &id1,
				Label:      &label1,
				Values:     []float64{value1},
				Timestamps: []time.Time{timestamp},
			},
			{
				Id:         &id2,
				Label:      &label2,
				Values:     []float64{value2},
				Timestamps: []time.Time{timestamp},
			},
		},
		NextToken:      &emptyString,
		ResultMetadata: middleware.Metadata{},
	}, nil
}

//func (m *MockCloudWatchClient) GetMetricDataRequest(input *cloudwatch.GetMetricDataInput) cloudwatch.GetMetricDataRequest {
//	httpReq, _ := http.NewRequest("", "", nil)
//
//	return cloudwatch.GetMetricDataRequest{
//		Input: input,
//		Copy:  m.GetMetricDataRequest,
//		Request: &awssdk.Request{
//			Operation: &awssdk.Operation{
//				Name:      "GetMetricData",
//				Paginator: nil,
//			},
//			Data: &cloudwatch.GetMetricDataOutput{

//			},
//			HTTPRequest: httpReq,
//			Retryer:     awssdk.NoOpRetryer{},
//		},
//	}
//}

//func (m *MockCloudWatchClient) ListMetricsRequest(input *cloudwatch.ListMetricsInput) cloudwatch.ListMetricsRequest {
//	dim := cloudwatchtypes.Dimension{
//		Name:  &dimName,
//		Value: &instanceID1,
//	}
//	httpReq, _ := http.NewRequest("", "", nil)
//	return cloudwatch.ListMetricsRequest{
//		Request: &awssdk.Request{
//			Data: &cloudwatch.ListMetricsOutput{
//				Metrics: []cloudwatchtypes.Metric{
//					{
//						MetricName: &metricName1,
//						Namespace:  &namespace,
//						Dimensions: []cloudwatchtypes.Dimension{dim},
//					},
//				},
//			},
//			HTTPRequest: httpReq,
//			Retryer:     awssdk.NoOpRetryer{},
//		},
//	}
//}

// MockCloudWatchClientWithoutDim struct is used for unit tests.
type MockCloudWatchClientWithoutDim struct {
	//*cloudwatch.Client
}

// GetMetricData implements cloudwatch.GetMetricDataAPIClient.
func (m *MockCloudWatchClientWithoutDim) GetMetricData(context.Context, *cloudwatch.GetMetricDataInput, ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
	emptyString := ""
	return &cloudwatch.GetMetricDataOutput{
		Messages: nil,
		MetricDataResults: []cloudwatchtypes.MetricDataResult{
			{
				Id:         &id1,
				Label:      &label3,
				Values:     []float64{value1},
				Timestamps: []time.Time{timestamp},
			},
			{
				Id:         &id2,
				Label:      &label4,
				Values:     []float64{value2},
				Timestamps: []time.Time{timestamp},
			},
		},
		NextToken:      &emptyString,
		ResultMetadata: middleware.Metadata{},
	}, nil
}

//func (m *MockCloudWatchClientWithoutDim) ListMetricsRequest(input *cloudwatch.ListMetricsInput) cloudwatch.ListMetricsRequest {
//	httpReq, _ := http.NewRequest("", "", nil)
//	return cloudwatch.ListMetricsRequest{
//		Request: &awssdk.Request{
//			Data: &cloudwatch.ListMetricsOutput{
//				Metrics: []cloudwatchtypes.Metric{
//					{
//						MetricName: &metricName1,
//						Namespace:  &namespace,
//					},
//				},
//			},
//			HTTPRequest: httpReq,
//			Retryer:     awssdk.NoOpRetryer{},
//		},
//	}
//}

//func (m *MockCloudWatchClientWithoutDim) GetMetricDataRequest(input *cloudwatch.GetMetricDataInput) cloudwatch.GetMetricDataRequest {
//	httpReq, _ := http.NewRequest("", "", nil)
//
//	return cloudwatch.GetMetricDataRequest{
//		Input: input,
//		Copy:  m.GetMetricDataRequest,
//		Request: &awssdk.Request{
//			Operation: &awssdk.Operation{
//				Name:      "GetMetricData",
//				Paginator: nil,
//			},
//			Data: &cloudwatch.GetMetricDataOutput{
//				MetricDataResults: []cloudwatchtypes.MetricDataResult{
//					{
//						Id:         &id1,
//						Label:      &label3,
//						Values:     []float64{value1},
//						Timestamps: []time.Time{timestamp},
//					},
//					{
//						Id:         &id2,
//						Label:      &label4,
//						Values:     []float64{value2},
//						Timestamps: []time.Time{timestamp},
//					},
//				},
//			},
//			HTTPRequest: httpReq,
//			Retryer:     awssdk.NoOpRetryer{},
//		},
//	}
//}

// MockResourceGroupsTaggingClient is used for unit tests.
type MockResourceGroupsTaggingClient struct {
	//*resourcegroupstaggingapi.Client
}

// GetResources implements resourcegroupstaggingapi.GetResourcesAPIClient.
func (m *MockResourceGroupsTaggingClient) GetResources(context.Context, *resourcegroupstaggingapi.GetResourcesInput, ...func(*resourcegroupstaggingapi.Options)) (*resourcegroupstaggingapi.GetResourcesOutput, error) {
	return &resourcegroupstaggingapi.GetResourcesOutput{
		PaginationToken: awssdk.String(""),
		ResourceTagMappingList: []resourcegroupstaggingapitypes.ResourceTagMapping{
			{
				ResourceARN: awssdk.String("arn:aws:ec2:us-west-1:123456789012:instance:i-1"),
				Tags: []resourcegroupstaggingapitypes.Tag{
					{
						Key:   awssdk.String("name"),
						Value: awssdk.String("test-ec2"),
					},
				},
			},
		},
		ResultMetadata: middleware.Metadata{},
	}, nil
}

//func (m *MockResourceGroupsTaggingClient) GetResourcesRequest(input *resourcegroupstaggingapi.GetResourcesInput) resourcegroupstaggingapi.GetResourcesRequest {
//	httpReq, _ := http.NewRequest("", "", nil)
//	return resourcegroupstaggingapi.GetResourcesRequest{
//		Input: input,
//		Copy:  m.GetResourcesRequest,
//		Request: &awssdk.Request{
//			Operation: &awssdk.Operation{
//				Name:       "GetResources",
//				HTTPMethod: "POST",
//				HTTPPath:   "/",
//				Paginator:  nil,
//			},
//			Data: &resourcegroupstaggingapi.GetResourcesOutput{
//				PaginationToken: awssdk.String(""),
//				ResourceTagMappingList: []resourcegroupstaggingapi.ResourceTagMapping{
//					{
//						ResourceARN: awssdk.String("arn:aws:ec2:us-west-1:123456789012:instance:i-1"),
//						Tags: []resourcegroupstaggingapi.Tag{
//							{
//								Key:   awssdk.String("name"),
//								Value: awssdk.String("test-ec2"),
//							},
//						},
//					},
//				},
//			},
//			HTTPRequest: httpReq,
//			Retryer:     awssdk.NoOpRetryer{},
//		},
//	}
//}

func TestCreateEventsWithIdentifier(t *testing.T) {
	m := MetricSet{}
	m.CloudwatchConfigs = []Config{{Statistic: []string{"Average"}}}
	m.MetricSet = &aws.MetricSet{Period: 5}
	m.logger = logp.NewLogger("test")

	mockTaggingSvc := &MockResourceGroupsTaggingClient{}
	mockCloudwatchSvc := &MockCloudWatchClient{}
	listMetricWithStatsTotal := []metricsWithStatistics{{
		cloudwatchtypes.Metric{
			Dimensions: []cloudwatchtypes.Dimension{{
				Name:  awssdk.String("InstanceId"),
				Value: awssdk.String("i-1"),
			}},
			MetricName: awssdk.String("CPUUtilization"),
			Namespace:  awssdk.String("AWS/EC2"),
		},
		[]string{"Average"},
		nil,
	}}
	resourceTypeTagFilters := map[string][]aws.Tag{}
	resourceTypeTagFilters["ec2:instance"] = []aws.Tag{
		{
			Key:   "name",
			Value: "test-ec2",
		},
	}
	startTime, endTime := aws.GetStartTimeEndTime(m.MetricSet.Period, m.MetricSet.Latency)

	events, err := m.createEvents(mockCloudwatchSvc, mockTaggingSvc, listMetricWithStatsTotal, resourceTypeTagFilters, regionName, startTime, endTime)
	assert.NoError(t, err)

	metricValue, err := events["i-1"].RootFields.GetValue("aws.ec2.metrics.CPUUtilization.avg")
	assert.NoError(t, err)
	assert.Equal(t, value1, metricValue)

	dimension, err := events["i-1"].RootFields.GetValue("aws.dimensions.InstanceId")
	assert.NoError(t, err)
	assert.Equal(t, instanceID1, dimension)
}

func TestCreateEventsWithoutIdentifier(t *testing.T) {
	m := MetricSet{}
	m.CloudwatchConfigs = []Config{{Statistic: []string{"Average"}}}
	m.MetricSet = &aws.MetricSet{Period: 5, AccountID: accountID}
	m.logger = logp.NewLogger("test")

	mockTaggingSvc := &MockResourceGroupsTaggingClient{}
	mockCloudwatchSvc := &MockCloudWatchClientWithoutDim{}
	listMetricWithStatsTotal := []metricsWithStatistics{
		{
			cloudwatchtypes.Metric{
				MetricName: awssdk.String("CPUUtilization"),
				Namespace:  awssdk.String("AWS/EC2"),
			},
			[]string{"Average"},
			nil,
		},
		{
			cloudwatchtypes.Metric{
				MetricName: awssdk.String("DiskReadOps"),
				Namespace:  awssdk.String("AWS/EC2"),
			},
			[]string{"Average"},
			nil,
		},
	}

	resourceTypeTagFilters := map[string][]aws.Tag{}
	startTime, endTime := aws.GetStartTimeEndTime(m.MetricSet.Period, m.MetricSet.Latency)

	events, err := m.createEvents(mockCloudwatchSvc, mockTaggingSvc, listMetricWithStatsTotal, resourceTypeTagFilters, regionName, startTime, endTime)
	assert.NoError(t, err)

	expectedID := regionName + accountID + namespace
	metricValue, err := events[expectedID].RootFields.GetValue("aws.ec2.metrics.CPUUtilization.avg")
	assert.NoError(t, err)
	assert.Equal(t, value1, metricValue)

	dimension, err := events[expectedID].RootFields.GetValue("aws.ec2.metrics.DiskReadOps.avg")
	assert.NoError(t, err)
	assert.Equal(t, value2, dimension)
}

func TestCreateEventsWithTagsFilter(t *testing.T) {
	m := MetricSet{}
	m.CloudwatchConfigs = []Config{{Statistic: []string{"Average"}}}
	m.MetricSet = &aws.MetricSet{Period: 5}
	m.logger = logp.NewLogger("test")

	mockTaggingSvc := &MockResourceGroupsTaggingClient{}
	mockCloudwatchSvc := &MockCloudWatchClient{}
	listMetricWithStatsTotal := []metricsWithStatistics{
		{
			cloudwatchtypes.Metric{
				Dimensions: []cloudwatchtypes.Dimension{{
					Name:  awssdk.String("InstanceId"),
					Value: awssdk.String("i-1"),
				}},
				MetricName: awssdk.String("CPUUtilization"),
				Namespace:  awssdk.String("AWS/EC2"),
			},
			[]string{"Average"},
			[]aws.Tag{
				{Key: "name", Value: "test-ec2"},
			},
		},
	}

	// Specify a tag filter that does not match the tag for i-1
	resourceTypeTagFilters := map[string][]aws.Tag{}
	resourceTypeTagFilters["ec2:instance"] = []aws.Tag{
		{
			Key:   "name",
			Value: "foo",
		},
	}
	startTime, endTime := aws.GetStartTimeEndTime(m.MetricSet.Period, m.MetricSet.Latency)

	events, err := m.createEvents(mockCloudwatchSvc, mockTaggingSvc, listMetricWithStatsTotal, resourceTypeTagFilters, regionName, startTime, endTime)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(events))
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

	resourceTagMap := map[string][]resourcegroupstaggingapitypes.Tag{}
	resourceTagMap["test-s3-1"] = []resourcegroupstaggingapitypes.Tag{
		{
			Key:   awssdk.String(tagKey1),
			Value: awssdk.String(tagValue1),
		},
	}
	resourceTagMap["test-s3-2"] = []resourcegroupstaggingapitypes.Tag{
		{
			Key:   awssdk.String(tagKey2),
			Value: awssdk.String(tagValue2),
		},
	}
	resourceTagMap["eipalloc-0123456789abcdef"] = []resourcegroupstaggingapitypes.Tag{
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
	cases := []struct {
		title          string
		dimensions     []Dimension
		expectedResult bool
	}{
		{
			"test dimensions without wolidcard value",
			[]Dimension{
				{
					Name:  "InstanceId",
					Value: "i-111111",
				},
				{
					Name:  "InstanceId",
					Value: "i-2222",
				},
			},
			false,
		},
		{
			"test dimensions without wolidcard value",
			[]Dimension{
				{
					Name:  "InstanceId",
					Value: "i-111111",
				},
				{
					Name:  "InstanceId",
					Value: dimensionValueWildcard,
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

func TestCreateEventsTimestamp(t *testing.T) {
	m := MetricSet{
		logger:            logp.NewLogger("test"),
		CloudwatchConfigs: []Config{{Statistic: []string{"Average"}}},
		MetricSet:         &aws.MetricSet{Period: 5, AccountID: accountID},
	}

	listMetricWithStatsTotal := []metricsWithStatistics{
		{
			cloudwatchtypes.Metric{
				MetricName: awssdk.String("CPUUtilization"),
				Namespace:  awssdk.String("AWS/EC2"),
			},
			[]string{"Average"},
			nil,
		},
	}

	resourceTypeTagFilters := map[string][]aws.Tag{}
	startTime, endTime := aws.GetStartTimeEndTime(m.MetricSet.Period, m.MetricSet.Latency)

	cloudwatchMock := &MockCloudWatchClientWithoutDim{}
	resGroupTaggingClientMock := &MockResourceGroupsTaggingClient{}
	events, err := m.createEvents(cloudwatchMock, resGroupTaggingClientMock, listMetricWithStatsTotal, resourceTypeTagFilters, regionName, startTime, endTime)
	assert.NoError(t, err)
	assert.Equal(t, timestamp, events[regionName+accountID+namespace].Timestamp)
}

func TestGetStartTimeEndTime(t *testing.T) {
	m := MetricSet{}
	m.CloudwatchConfigs = []Config{{Statistic: []string{"Average"}}}
	m.MetricSet = &aws.MetricSet{Period: 5 * time.Minute}
	m.logger = logp.NewLogger("test")
	startTime, endTime := aws.GetStartTimeEndTime(m.MetricSet.Period, m.MetricSet.Latency)
	assert.Equal(t, 5*time.Minute, endTime.Sub(startTime))
}
