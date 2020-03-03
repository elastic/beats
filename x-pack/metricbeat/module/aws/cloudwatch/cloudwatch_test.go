// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package cloudwatch

import (
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
)

var (
	instanceID1     = "i-1"
	instanceID2     = "i-2"
	namespace       = "AWS/EC2"
	dimName         = "InstanceId"
	metricName1     = "CPUUtilization"
	metricName2     = "StatusCheckFailed"
	metricName3     = "StatusCheckFailed_System"
	metricName4     = "StatusCheckFailed_Instance"
	resourceTypeEC2 = "ec2:instance"
	listMetric1     = cloudwatch.Metric{
		Dimensions: []cloudwatch.Dimension{{
			Name:  &dimName,
			Value: &instanceID1,
		}},
		MetricName: &metricName1,
		Namespace:  &namespace,
	}

	listMetric2 = cloudwatch.Metric{
		Dimensions: []cloudwatch.Dimension{{
			Name:  &dimName,
			Value: &instanceID1,
		}},
		MetricName: &metricName2,
		Namespace:  &namespace,
	}

	listMetric3 = cloudwatch.Metric{
		Dimensions: []cloudwatch.Dimension{{
			Name:  &dimName,
			Value: &instanceID2,
		}},
		MetricName: &metricName3,
		Namespace:  &namespace,
	}

	listMetric4 = cloudwatch.Metric{
		Dimensions: []cloudwatch.Dimension{{
			Name:  &dimName,
			Value: &instanceID2,
		}},
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

func TestReadCloudwatchConfig(t *testing.T) {
	m := MetricSet{}
	resourceTypeFiltersEC2 := map[string][]aws.Tag{}
	resourceTypeFiltersEC2["ec2:instance"] = nil

	expectedListMetricWithDetailEC2 := listMetricWithDetail{
		metricsWithStats: []metricsWithStatistics{
			{
				cloudwatch.Metric{
					Dimensions: []cloudwatch.Dimension{{
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
				cloudwatch.Metric{
					Dimensions: []cloudwatch.Dimension{{
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
				cloudwatch.Metric{
					Dimensions: []cloudwatch.Dimension{{
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
					Value: "test-ec2",
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
	}

	expectedNamespaceDetailELBLambda := map[string][]namespaceDetail{}
	expectedNamespaceDetailELBLambda["AWS/Lambda"] = []namespaceDetail{
		{
			statistics: defaultStatistics,
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
	}

	expectedNamespaceWithDetailEC2WithNoMetricName := map[string][]namespaceDetail{}
	expectedNamespaceWithDetailEC2WithNoMetricName["AWS/EC2"] = []namespaceDetail{
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
	}

	expectedListMetricsEC2WithDim := listMetricWithDetail{
		metricsWithStats: []metricsWithStatistics{
			{
				cloudwatch.Metric{
					Dimensions: []cloudwatch.Dimension{{
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
				cloudwatch.Metric{
					Dimensions: []cloudwatch.Dimension{{
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
					ResourceTypeFilter: "ec2:instance",
					Statistic:          []string{"Average"},
				},
			},
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
					ResourceTypeFilter: "ec2:instance",
					Statistic:          []string{"Average"},
				},
				{
					Namespace: "AWS/S3",
				},
			},
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
					Statistic:          []string{"Average"},
					ResourceTypeFilter: "ec2:instance",
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
					Statistic:          []string{"Average"},
					ResourceTypeFilter: "rds",
				},
			},
			expectedListMetricWithDetailEC2RDS,
			expectedNamespaceDetailLambda,
		},
		{
			"Test a specific metric (only with metric name) and a namespace",
			[]Config{
				{
					Namespace:          "AWS/EC2",
					MetricName:         []string{"CPUUtilization"},
					ResourceTypeFilter: resourceTypeEC2,
				},
				{
					Namespace:          "AWS/S3",
					ResourceTypeFilter: "s3",
				},
			},
			listMetricWithDetail{
				resourceTypeFilters: map[string][]aws.Tag{},
			},
			expectedNamespaceWithDetailEC2S3,
		},
		{
			"test EBS namespace",
			[]Config{
				{
					Namespace:          "AWS/EBS",
					ResourceTypeFilter: "ec2",
				},
			},
			listMetricWithDetail{
				resourceTypeFilters: map[string][]aws.Tag{},
			},
			expectedNamespaceWithDetailEBS,
		},
		{
			"test with two metrics and no dimension",
			[]Config{
				{
					Namespace:          "AWS/EC2",
					MetricName:         []string{"CPUUtilization", "StatusCheckFailed"},
					ResourceTypeFilter: resourceTypeEC2,
					Statistic:          []string{"Average", "Maximum"},
				},
			},
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
			listMetricWithDetail{
				resourceTypeFilters: map[string][]aws.Tag{},
			},
			expectedNamespaceWithDetailKafka,
		},
		{
			"Test with different statistics",
			[]Config{
				{
					Namespace:          "AWS/EC2",
					MetricName:         []string{"CPUUtilization"},
					ResourceTypeFilter: resourceTypeEC2,
					TagsFilter: []aws.Tag{
						{
							Key:   "name",
							Value: "test-ec2",
						},
					},
				},
				{
					Namespace:          "AWS/ELB",
					MetricName:         []string{"BackendConnectionErrors", "HTTPCode_Backend_2XX", "HTTPCode_Backend_3XX"},
					Statistic:          []string{"Sum"},
					ResourceTypeFilter: "elasticloadbalancing",
					TagsFilter: []aws.Tag{
						{
							Key:   "name",
							Value: "test-elb1",
						},
					},
				},
				{
					Namespace:          "AWS/ELB",
					MetricName:         []string{"HealthyHostCount", "SurgeQueueLength", "UnHealthyHostCount"},
					Statistic:          []string{"Maximum"},
					ResourceTypeFilter: "elasticloadbalancing",
					TagsFilter: []aws.Tag{
						{
							Key:   "name",
							Value: "test-elb2",
						},
					},
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
					Namespace:          "AWS/ELB",
					MetricName:         []string{"BackendConnectionErrors", "HTTPCode_Backend_2XX", "HTTPCode_Backend_3XX"},
					Statistic:          []string{"Sum"},
					ResourceTypeFilter: "elasticloadbalancing",
					TagsFilter: []aws.Tag{
						{
							Key:   "name",
							Value: "test-elb1",
						},
					},
				},
				{
					Namespace:          "AWS/ELB",
					MetricName:         []string{"HealthyHostCount", "SurgeQueueLength", "UnHealthyHostCount"},
					Statistic:          []string{"Maximum"},
					ResourceTypeFilter: "elasticloadbalancing",
					TagsFilter: []aws.Tag{
						{
							Key:   "name",
							Value: "test-elb2",
						},
					},
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
					Statistic:          []string{"Average"},
					ResourceTypeFilter: "ec2:instance",
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
					Statistic:          []string{"Average"},
					ResourceTypeFilter: "rds",
				},
			},
			expectedListMetricWithDetailEC2RDS,
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
					Statistic:          []string{"Average"},
					ResourceTypeFilter: "ec2:instance",
				},
			},
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
					ResourceTypeFilter: "ec2:instance",
					Statistic:          []string{"Average"},
				},
			},
			expectedListMetricsEC2WithDim,
			map[string][]namespaceDetail{},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			m.CloudwatchConfigs = c.cloudwatchMetricsConfig
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
		listMetricsOutput            []cloudwatch.Metric
		namespaceDetails             []namespaceDetail
		filteredMetricWithStatsTotal []metricsWithStatistics
	}{
		{
			"test filter cloudwatch metrics with dimension",
			[]cloudwatch.Metric{
				{
					Dimensions: []cloudwatch.Dimension{
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
					Dimensions: []cloudwatch.Dimension{{
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
						Dimensions: []cloudwatch.Dimension{{
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
			[]cloudwatch.Metric{
				{
					Dimensions: []cloudwatch.Dimension{
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
					Dimensions: []cloudwatch.Dimension{{
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
					cloudwatch.Metric{
						Dimensions: []cloudwatch.Dimension{{
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
