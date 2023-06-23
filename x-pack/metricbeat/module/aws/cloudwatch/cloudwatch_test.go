// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package cloudwatch

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	cloudwatchtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	resourcegroupstaggingapitypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
	"github.com/aws/smithy-go/middleware"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	regionName  = "us-west-1"
	timestamp   = time.Date(2020, 10, 06, 00, 00, 00, 0, time.UTC)
	accountID   = "123456789012"
	accountName = "test"

	id1    = "cpu"
	value1 = 0.25
	label1 = " | |CPUUtilization|AWS/EC2|Average|300|InstanceId|i-1"

	id2    = "disk"
	value2 = 5.0
	label2 = " | |DiskReadOps|AWS/EC2|Average|300|InstanceId|i-1"

	label3 = " | |CPUUtilization|AWS/EC2|Average|300"
	label4 = " | |DiskReadOps|AWS/EC2|Average|300"

	instanceID1     = "i-1"
	instanceID2     = "i-2"
	namespace       = "AWS/EC2"
	dimName         = "InstanceId"
	metricName1     = "CPUUtilization"
	metricName2     = "StatusCheckFailed"
	metricName3     = "StatusCheckFailed_System"
	metricName4     = "StatusCheckFailed_Instance"
	metricName5     = "DiskReadOps"
	resourceTypeEC2 = "ec2:instance"
	listMetric1     = aws.MetricWithID{
		Metric: cloudwatchtypes.Metric{
			Dimensions: []cloudwatchtypes.Dimension{{
				Name:  &dimName,
				Value: &instanceID1,
			}},
			MetricName: &metricName1,
			Namespace:  &namespace,
		},
	}

	listMetric2 = aws.MetricWithID{
		Metric: cloudwatchtypes.Metric{
			Dimensions: []cloudwatchtypes.Dimension{{
				Name:  &dimName,
				Value: &instanceID1,
			}},
			MetricName: &metricName2,
			Namespace:  &namespace,
		},
		AccountID: accountID,
	}

	listMetric3 = aws.MetricWithID{
		Metric: cloudwatchtypes.Metric{
			Dimensions: []cloudwatchtypes.Dimension{{
				Name:  &dimName,
				Value: &instanceID2,
			}},
			MetricName: &metricName3,
			Namespace:  &namespace,
		},
		AccountID: accountID,
	}

	listMetric4 = aws.MetricWithID{
		Metric: cloudwatchtypes.Metric{
			Dimensions: []cloudwatchtypes.Dimension{{
				Name:  &dimName,
				Value: &instanceID2,
			}},
			MetricName: &metricName4,
			Namespace:  &namespace,
		},
		AccountID: accountID,
	}

	listMetric5 = aws.MetricWithID{
		Metric: cloudwatchtypes.Metric{
			MetricName: &metricName1,
			Namespace:  &namespace,
		},
		AccountID: accountID,
	}

	listMetric6 = aws.MetricWithID{
		Metric: cloudwatchtypes.Metric{
			Dimensions: []cloudwatchtypes.Dimension{{
				Name:  &dimName,
				Value: &instanceID1,
			}},
			MetricName: &metricName5,
			Namespace:  &namespace,
		},
	}

	namespaceMSK = "AWS/Kafka"
	metricName6  = "MemoryUsed"
	listMetric8  = aws.MetricWithID{
		Metric: cloudwatchtypes.Metric{
			MetricName: &metricName6,
			Namespace:  &namespaceMSK,
		},
		AccountID: accountID,
	}
	nameTestTag = []aws.Tag{
		{
			Key:   "name",
			Value: []string{"test"},
		}}
	nameTestEC2Tag = []aws.Tag{
		{
			Key:   "name",
			Value: []string{"test-ec2"},
		}}
	nameELBTag = []aws.Tag{
		{
			Key:   "name",
			Value: []string{"test-elb"},
		},
	}
	nameELB1Tag = []aws.Tag{
		{
			Key:   "name",
			Value: []string{"test-elb1"},
		},
	}
	nameELB2Tag = []aws.Tag{
		{
			Key:   "name",
			Value: []string{"test-elb2"},
		},
	}
	elbNamespaceDetail = []namespaceDetail{
		{
			resourceTypeFilter: "elasticloadbalancing",
			names:              []string{"BackendConnectionErrors", "HTTPCode_Backend_2XX", "HTTPCode_Backend_3XX"},
			statistics:         []string{"Sum"},
			tags:               nameTestTag,
		},
		{
			resourceTypeFilter: "elasticloadbalancing",
			names:              []string{"HealthyHostCount", "SurgeQueueLength", "UnHealthyHostCount"},
			statistics:         []string{"Maximum"},
			tags:               nameTestTag,
		},
	}
	ec2instance1Config = Config{
		Namespace:  namespace,
		MetricName: []string{metricName1},
		Dimensions: []Dimension{
			{
				Name:  "InstanceId",
				Value: instanceID1,
			},
		},
		ResourceType: resourceTypeEC2,
		Statistic:    []string{"Average"},
	}
)

func TestConstructLabel(t *testing.T) {
	cases := []struct {
		listMetricDetail aws.MetricWithID
		statistic        string
		expectedLabel    string
	}{
		{
			listMetric1,
			"Average",
			"|${PROP('AccountLabel')}|CPUUtilization|AWS/EC2|Average|${PROP('Period')}|InstanceId|i-1",
		},
		{
			listMetric2,
			"Maximum",
			"123456789012|${PROP('AccountLabel')}|StatusCheckFailed|AWS/EC2|Maximum|${PROP('Period')}|InstanceId|i-1",
		},
		{
			listMetric3,
			"Minimum",
			"123456789012|${PROP('AccountLabel')}|StatusCheckFailed_System|AWS/EC2|Minimum|${PROP('Period')}|InstanceId|i-2",
		},
		{
			listMetric4,
			"Sum",
			"123456789012|${PROP('AccountLabel')}|StatusCheckFailed_Instance|AWS/EC2|Sum|${PROP('Period')}|InstanceId|i-2",
		},
		{
			listMetric5,
			"SampleCount",
			"123456789012|${PROP('AccountLabel')}|CPUUtilization|AWS/EC2|SampleCount|${PROP('Period')}",
		},
		{
			listMetric8,
			"SampleCount",
			"123456789012|${PROP('AccountLabel')}|MemoryUsed|AWS/Kafka|SampleCount|${PROP('Period')}",
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
				listMetric1,
				[]string{"Average"},
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
	metricsWithStats := []metricsWithStatistics{
		{
			listMetric1,
			[]string{"Average"},
		},
		{
			aws.MetricWithID{
				Metric: cloudwatchtypes.Metric{
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
			},
			[]string{"Average"},
		},
	}

	expectedListMetricWithDetailEC2RDS := listMetricWithDetail{
		metricsWithStats:    metricsWithStats,
		resourceTypeFilters: resourceTypeFiltersEC2RDS,
	}

	resourceTypeFiltersEC2RDSWithTag := map[string][]aws.Tag{}
	resourceTypeFiltersEC2RDSWithTag["ec2:instance"] = nameTestTag
	resourceTypeFiltersEC2RDSWithTag["rds"] = nameTestTag
	expectedListMetricWithDetailEC2RDSWithTag := listMetricWithDetail{
		metricsWithStats:    metricsWithStats,
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
			tags:               nameTestTag,
		},
	}

	expectedNamespaceDetailTotal["AWS/ELB"] = elbNamespaceDetail

	expectedNamespaceDetailELBLambda := map[string][]namespaceDetail{}
	expectedNamespaceDetailELBLambda["AWS/Lambda"] = []namespaceDetail{
		{
			statistics: defaultStatistics,
			tags:       nameTestTag,
		},
	}
	expectedNamespaceDetailELBLambda["AWS/ELB"] = elbNamespaceDetail

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
				listMetric1,
				[]string{"Average"},
			},
			{
				listMetric6,
				[]string{"Average"},
			},
		},
		resourceTypeFilters: resourceTypeFiltersEC2,
	}

	expectedListMetricWithDetailEC2sRDSWithTag := listMetricWithDetail{
		metricsWithStats: []metricsWithStatistics{
			{
				listMetric1,
				[]string{"Average"},
			},
			{
				aws.MetricWithID{
					Metric: cloudwatchtypes.Metric{
						Dimensions: []cloudwatchtypes.Dimension{{
							Name:  awssdk.String("InstanceId"),
							Value: awssdk.String("i-2"),
						}},
						MetricName: awssdk.String("DiskReadBytes"),
						Namespace:  awssdk.String("AWS/EC2"),
					},
				},
				[]string{"Sum"},
			},
			{
				aws.MetricWithID{
					Metric: cloudwatchtypes.Metric{
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
				},
				[]string{"Average"},
			},
		},
		resourceTypeFilters: resourceTypeFiltersEC2RDSWithTag,
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
				ec2instance1Config,
			},
			nil,
			expectedListMetricWithDetailEC2,
			map[string][]namespaceDetail{},
		},
		{
			"test with a specific metric and a namespace",
			[]Config{
				ec2instance1Config,
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
				ec2instance1Config,
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
			nameTestTag,
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
				ec2instance1Config,
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
			nameTestTag,
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
		{
			"Test with same namespace and tag filters but different metric names",
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
					Namespace:  "AWS/EC2",
					MetricName: []string{"DiskReadBytes"},
					Dimensions: []Dimension{
						{
							Name:  "InstanceId",
							Value: "i-2",
						},
					},
					Statistic:    []string{"Sum"},
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
			nameTestTag,
			expectedListMetricWithDetailEC2sRDSWithTag,
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
			[]string{"", "${PROP('AccountLabel')}", "CPUUtilization", "AWS/EC2", "Average", "InstanceId", "i-1"},
			"aws.ec2.metrics.CPUUtilization.avg",
		},
		{
			"test Maximum",
			"cloudwatch",
			[]string{"", "${PROP('AccountLabel')}", "CPUUtilization", "AWS/EC2", "Maximum", "InstanceId", "i-1"},
			"aws.ec2.metrics.CPUUtilization.max",
		},
		{
			"test Minimum",
			"cloudwatch",
			[]string{"", "${PROP('AccountLabel')}", "CPUUtilization", "AWS/EC2", "Minimum", "InstanceId", "i-1"},
			"aws.ec2.metrics.CPUUtilization.min",
		},
		{
			"test Sum",
			"cloudwatch",
			[]string{"", "${PROP('AccountLabel')}", "CPUUtilization", "AWS/EC2", "Sum", "InstanceId", "i-1"},
			"aws.ec2.metrics.CPUUtilization.sum",
		},
		{
			"test SampleCount",
			"cloudwatch",
			[]string{"", "${PROP('AccountLabel')}", "CPUUtilization", "AWS/EC2", "SampleCount", "InstanceId", "i-1"},
			"aws.ec2.metrics.CPUUtilization.count",
		},
		{
			"test extended statistic",
			"cloudwatch",
			[]string{"", "${PROP('AccountLabel')}", "CPUUtilization", "AWS/EC2", "p10", "InstanceId", "i-1"},
			"aws.ec2.metrics.CPUUtilization.p10",
		},
		{
			"test other metricset",
			"ec2",
			[]string{"", "${PROP('AccountLabel')}", "CPUUtilization", "AWS/EC2", "p10", "InstanceId", "i-1"},
			"aws.ec2.metrics.CPUUtilization.p10",
		},
		{
			"test metric name with dot",
			"cloudwatch",
			[]string{"", "${PROP('AccountLabel')}", "DeliveryToS3.Records", "AWS/Firehose", "Average", "DeliveryStreamName", "test-1"},
			"aws.firehose.metrics.DeliveryToS3_Records.avg",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			fieldName := generateFieldName(c.label[aws.LabelConst.NamespaceIdx], c.label)
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
	expectedResourceTypeTagFiltersELB["elasticloadbalancing"] = append(nameELB1Tag, nameELB2Tag...)

	expectedResourceTypeTagFiltersELBEC2 := map[string][]aws.Tag{}
	expectedResourceTypeTagFiltersELBEC2["elasticloadbalancing"] = nameELBTag
	expectedResourceTypeTagFiltersELBEC2["ec2:instance"] = nameTestEC2Tag

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
					tags:               nameELB1Tag,
				},
				{
					resourceTypeFilter: "elasticloadbalancing",
					names:              []string{"HealthyHostCount", "SurgeQueueLength", "UnHealthyHostCount"},
					statistics:         []string{"Maximum"},
					tags:               nameELB2Tag,
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
					tags:               nameELBTag,
				},
				{
					resourceTypeFilter: "ec2:instance",
					names:              []string{"CPUUtilization"},
					statistics:         defaultStatistics,
					tags:               nameTestEC2Tag,
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
		listMetricsOutput            []aws.MetricWithID
		namespace                    string
		namespaceDetails             []namespaceDetail
		filteredMetricWithStatsTotal []metricsWithStatistics
	}{
		{
			"test filter cloudwatch metrics with dimension",
			[]aws.MetricWithID{
				{
					Metric: cloudwatchtypes.Metric{
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
				},
				{
					Metric: cloudwatchtypes.Metric{
						Dimensions: []cloudwatchtypes.Dimension{{
							Name:  awssdk.String("InstanceId"),
							Value: awssdk.String("i-1"),
						}},
						MetricName: awssdk.String("CPUUtilization"),
						Namespace:  awssdk.String("AWS/EC2"),
					},
				},
			},
			"AWS/EC2",
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
					listMetric1,
					[]string{"Average"},
				},
			},
		},
		{
			"test filter cloudwatch metrics with name",
			[]aws.MetricWithID{
				{
					Metric: cloudwatchtypes.Metric{
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
				},
				{
					Metric: cloudwatchtypes.Metric{
						Dimensions: []cloudwatchtypes.Dimension{{
							Name:  awssdk.String("InstanceId"),
							Value: awssdk.String("i-1"),
						}},
						MetricName: awssdk.String("CPUUtilization"),
						Namespace:  awssdk.String("AWS/EC2"),
					},
				},
			},
			"AWS/EC2",
			[]namespaceDetail{
				{
					names:              []string{"CPUUtilization"},
					resourceTypeFilter: "ec2:instance",
					statistics:         []string{"Average"},
				},
			},
			[]metricsWithStatistics{
				{
					listMetric1,
					[]string{"Average"},
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			output := filterListMetricsOutput(c.listMetricsOutput, c.namespace, c.namespaceDetails)
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
type MockCloudWatchClient struct{}

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

// MockCloudWatchClientWithoutDim struct is used for unit tests.
type MockCloudWatchClientWithoutDim struct{}

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

// MockCloudWatchClientWithDataGranularity struct is used for unit tests.
type MockCloudWatchClientWithDataGranularity struct{}

// GetMetricData implements cloudwatch.GetMetricDataAPIClient.
func (m *MockCloudWatchClientWithDataGranularity) GetMetricData(context.Context, *cloudwatch.GetMetricDataInput, ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
	emptyString := ""
	return &cloudwatch.GetMetricDataOutput{
		Messages: nil,
		MetricDataResults: []cloudwatchtypes.MetricDataResult{
			{
				Id:         &id1,
				Label:      &label3,
				Values:     []float64{value1, value1},
				Timestamps: []time.Time{timestamp, timestamp},
			},
			{
				Id:         &id2,
				Label:      &label4,
				Values:     []float64{value2, value2},
				Timestamps: []time.Time{timestamp, timestamp},
			},
		},
		NextToken:      &emptyString,
		ResultMetadata: middleware.Metadata{},
	}, nil
}

// MockResourceGroupsTaggingClient is used for unit tests.
type MockResourceGroupsTaggingClient struct{}

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

func TestCreateEventsWithIdentifier(t *testing.T) {
	m := MetricSet{}
	m.CloudwatchConfigs = []Config{{Statistic: []string{"Average"}}}
	m.MetricSet = &aws.MetricSet{Period: 5}
	m.logger = logp.NewLogger("test")

	mockTaggingSvc := &MockResourceGroupsTaggingClient{}
	mockCloudwatchSvc := &MockCloudWatchClient{}
	listMetricWithStatsTotal := []metricsWithStatistics{{
		listMetric1,
		[]string{"Average"},
	}}
	resourceTypeTagFilters := map[string][]aws.Tag{}
	resourceTypeTagFilters["ec2:instance"] = nameTestEC2Tag
	startTime, endTime := aws.GetStartTimeEndTime(time.Now(), m.MetricSet.Period, m.MetricSet.Latency)

	events, err := m.createEvents(mockCloudwatchSvc, mockTaggingSvc, listMetricWithStatsTotal, resourceTypeTagFilters, regionName, startTime, endTime)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))

	metricValue, err := events["i-1-0"].RootFields.GetValue("aws.ec2.metrics.CPUUtilization.avg")
	assert.NoError(t, err)
	assert.Equal(t, value1, metricValue)

	dimension, err := events["i-1-0"].RootFields.GetValue("aws.dimensions.InstanceId")
	assert.NoError(t, err)
	assert.Equal(t, instanceID1, dimension)
}

func TestCreateEventsWithoutIdentifier(t *testing.T) {
	m := MetricSet{}
	m.CloudwatchConfigs = []Config{{Statistic: []string{"Average"}}}
	m.MetricSet = &aws.MetricSet{Period: 5, MonitoringAccountID: accountID}
	m.logger = logp.NewLogger("test")

	mockTaggingSvc := &MockResourceGroupsTaggingClient{}
	mockCloudwatchSvc := &MockCloudWatchClientWithoutDim{}
	listMetricWithStatsTotal := []metricsWithStatistics{
		{
			cloudwatchMetric: aws.MetricWithID{
				Metric: cloudwatchtypes.Metric{
					MetricName: awssdk.String("CPUUtilization"),
					Namespace:  awssdk.String("AWS/EC2"),
				},
			},
			statistic: []string{"Average"},
		},
		{
			cloudwatchMetric: aws.MetricWithID{
				Metric: cloudwatchtypes.Metric{
					MetricName: awssdk.String("DiskReadOps"),
					Namespace:  awssdk.String("AWS/EC2"),
				},
			},
			statistic: []string{"Average"},
		},
	}

	resourceTypeTagFilters := map[string][]aws.Tag{}
	startTime, endTime := aws.GetStartTimeEndTime(time.Now(), m.MetricSet.Period, m.MetricSet.Latency)

	events, err := m.createEvents(mockCloudwatchSvc, mockTaggingSvc, listMetricWithStatsTotal, resourceTypeTagFilters, regionName, startTime, endTime)
	assert.NoError(t, err)

	expectedID := " " + " " + regionName + accountID + namespace
	metricValue, err := events[expectedID+"-0"].RootFields.GetValue("aws.ec2.metrics.CPUUtilization.avg")
	assert.NoError(t, err)
	assert.Equal(t, value1, metricValue)

	dimension, err := events[expectedID+"-0"].RootFields.GetValue("aws.ec2.metrics.DiskReadOps.avg")
	assert.NoError(t, err)
	assert.Equal(t, value2, dimension)
}

func TestCreateEventsWithDataGranularity(t *testing.T) {
	m := MetricSet{}
	m.CloudwatchConfigs = []Config{{Statistic: []string{"Average"}}}
	m.MetricSet = &aws.MetricSet{Period: 10, MonitoringAccountID: accountID, DataGranularity: 5}
	m.logger = logp.NewLogger("test")

	mockTaggingSvc := &MockResourceGroupsTaggingClient{}
	mockCloudwatchSvc := &MockCloudWatchClientWithDataGranularity{}
	listMetricWithStatsTotal := []metricsWithStatistics{
		{
			listMetric1,
			[]string{"Average"},
		},
		{
			listMetric6,
			[]string{"Average"},
		},
	}

	resourceTypeTagFilters := map[string][]aws.Tag{}
	startTime, endTime := aws.GetStartTimeEndTime(time.Now(), m.MetricSet.Period, m.MetricSet.Latency)

	events, err := m.createEvents(mockCloudwatchSvc, mockTaggingSvc, listMetricWithStatsTotal, resourceTypeTagFilters, regionName, startTime, endTime)
	assert.NoError(t, err)

	expectedID := "  " + regionName + accountID
	metricValue, err := events[expectedID+namespace+"-0"].RootFields.GetValue("aws.ec2.metrics.CPUUtilization.avg")
	assert.NoError(t, err)
	metricValue1, err := events[expectedID+namespace+"-1"].RootFields.GetValue("aws.ec2.metrics.CPUUtilization.avg")
	assert.NoError(t, err)
	metricValue2, err := events[expectedID+namespace+"-0"].RootFields.GetValue("aws.ec2.metrics.DiskReadOps.avg")
	assert.NoError(t, err)
	metricValue3, err := events[expectedID+namespace+"-1"].RootFields.GetValue("aws.ec2.metrics.DiskReadOps.avg")
	assert.NoError(t, err)
	assert.Equal(t, value1, metricValue)
	assert.Equal(t, value1, metricValue1)
	assert.Equal(t, value2, metricValue2)
	assert.Equal(t, value2, metricValue3)
	assert.Equal(t, 2, len(events))
}

func TestCreateEventsWithTagsFilter(t *testing.T) {
	m := MetricSet{}
	m.CloudwatchConfigs = []Config{{Statistic: []string{"Average"}}}
	m.MetricSet = &aws.MetricSet{Period: 5, MonitoringAccountID: accountID}
	m.logger = logp.NewLogger("test")

	mockTaggingSvc := &MockResourceGroupsTaggingClient{}
	mockCloudwatchSvc := &MockCloudWatchClient{}
	listMetricWithStatsTotal := []metricsWithStatistics{
		{
			listMetric1,
			[]string{"Average"},
		},
		{
			listMetric6,
			[]string{"Average"},
		},
	}

	// Check that the event is created when the tag filter matches
	resourceTypeTagFilters := map[string][]aws.Tag{}
	resourceTypeTagFilters["ec2:instance"] = nameTestEC2Tag

	startTime, endTime := aws.GetStartTimeEndTime(time.Now(), m.MetricSet.Period, m.MetricSet.Latency)
	events, err := m.createEvents(mockCloudwatchSvc, mockTaggingSvc, listMetricWithStatsTotal, resourceTypeTagFilters, regionName, startTime, endTime)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))

	// Specify a tag filter that does not match the tag for i-1
	resourceTypeTagFilters["ec2:instance"] = []aws.Tag{
		{
			Key:   "name",
			Value: []string{"foo"},
		},
	}

	events, err = m.createEvents(mockCloudwatchSvc, mockTaggingSvc, listMetricWithStatsTotal, resourceTypeTagFilters, regionName, startTime, endTime)
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
	cloudwatchPeriod := "300"

	events := map[string]mb.Event{}
	events[identifier1] = aws.InitEvent(regionName, accountName, accountID, timestamp, cloudwatchPeriod)
	events[identifier2] = aws.InitEvent(regionName, accountName, accountID, timestamp, cloudwatchPeriod)
	events[identifierContainsArn] = aws.InitEvent(regionName, accountName, accountID, timestamp, cloudwatchPeriod)

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
			subIdentifiers := strings.Split(c.identifier, dimensionSeparator)
			for _, subIdentifier := range subIdentifiers {
				insertTags(events, c.identifier, subIdentifier, resourceTagMap)
			}
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
		MetricSet:         &aws.MetricSet{Period: 5, MonitoringAccountID: accountID},
	}

	listMetricWithStatsTotal := []metricsWithStatistics{
		{
			listMetric1,
			[]string{"Average"},
		},
		{
			aws.MetricWithID{
				Metric: cloudwatchtypes.Metric{

					MetricName: awssdk.String("DiskReadOps"),
					Namespace:  awssdk.String("AWS/EC2"),
				},
			},
			[]string{"Average"},
		},
	}

	resourceTypeTagFilters := map[string][]aws.Tag{}
	startTime, endTime := aws.GetStartTimeEndTime(time.Now(), m.MetricSet.Period, m.MetricSet.Latency)

	cloudwatchMock := &MockCloudWatchClientWithoutDim{}
	resGroupTaggingClientMock := &MockResourceGroupsTaggingClient{}
	events, err := m.createEvents(cloudwatchMock, resGroupTaggingClientMock, listMetricWithStatsTotal, resourceTypeTagFilters, regionName, startTime, endTime)
	assert.NoError(t, err)
	assert.Equal(t, timestamp, events["  "+regionName+accountID+namespace+"-0"].Timestamp)
}

func TestGetStartTimeEndTime(t *testing.T) {
	m := MetricSet{}
	m.CloudwatchConfigs = []Config{{Statistic: []string{"Average"}}}
	m.MetricSet = &aws.MetricSet{Period: 5 * time.Minute}
	m.logger = logp.NewLogger("test")
	startTime, endTime := aws.GetStartTimeEndTime(time.Now(), m.MetricSet.Period, m.MetricSet.Latency)
	assert.Equal(t, 5*time.Minute, endTime.Sub(startTime))
}
