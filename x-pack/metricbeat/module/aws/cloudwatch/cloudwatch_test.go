// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package cloudwatch

import (
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/stretchr/testify/assert"
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
	metricName5     = "CommitThroughput"
	namespaceRDS    = "AWS/RDS"
	resourceTypeEC2 = "ec2:instance"
	resourceTypeRDS = "rds"
	listMetric1     = cloudwatch.Metric{
		Dimensions: []cloudwatch.Dimension{{
			Name:  &dimName,
			Value: &instanceID1,
		}},
		MetricName: &metricName1,
		Namespace:  &namespace,
	}

	metricsWithStat1 = metricsWithStatistics{
		listMetric1,
		[]string{"Average"},
	}

	listMetricWithDetail1 = listMetricWithDetail{
		metricsWithStats:    []metricsWithStatistics{metricsWithStat1},
		resourceTypeFilters: []string{resourceTypeEC2},
	}

	listMetric2 = cloudwatch.Metric{
		Dimensions: []cloudwatch.Dimension{{
			Name:  &dimName,
			Value: &instanceID1,
		}},
		MetricName: &metricName2,
		Namespace:  &namespace,
	}

	metricsWithStat2 = metricsWithStatistics{
		listMetric2,
		[]string{"Maximum"},
	}

	listMetricWithDetail2 = listMetricWithDetail{
		metricsWithStats:    []metricsWithStatistics{metricsWithStat2},
		resourceTypeFilters: []string{resourceTypeEC2},
	}

	listMetric3 = cloudwatch.Metric{
		Dimensions: []cloudwatch.Dimension{{
			Name:  &dimName,
			Value: &instanceID2,
		}},
		MetricName: &metricName3,
		Namespace:  &namespace,
	}

	metricsWithStat3 = metricsWithStatistics{
		listMetric3,
		[]string{"Minimum"},
	}

	listMetricWithDetail3 = listMetricWithDetail{
		metricsWithStats:    []metricsWithStatistics{metricsWithStat3},
		resourceTypeFilters: []string{resourceTypeEC2},
	}

	listMetric4 = cloudwatch.Metric{
		Dimensions: []cloudwatch.Dimension{{
			Name:  &dimName,
			Value: &instanceID2,
		}},
		MetricName: &metricName4,
		Namespace:  &namespace,
	}

	metricsWithStat4 = metricsWithStatistics{
		listMetric4,
		[]string{"Sum"},
	}

	listMetricWithDetail4 = listMetricWithDetail{
		metricsWithStats:    []metricsWithStatistics{metricsWithStat4},
		resourceTypeFilters: []string{resourceTypeEC2},
	}

	listMetric5 = cloudwatch.Metric{
		MetricName: &metricName1,
		Namespace:  &namespace,
	}

	metricsWithStat5 = metricsWithStatistics{
		listMetric5,
		[]string{"SampleCount"},
	}

	listMetricWithDetail5 = listMetricWithDetail{
		metricsWithStats:    []metricsWithStatistics{metricsWithStat5},
		resourceTypeFilters: []string{resourceTypeRDS},
	}

	dimName1    = "DBClusterIdentifier"
	dimValue1   = "test1-cluster"
	dimName2    = "Role"
	dimValue2   = "READER"
	listMetric6 = cloudwatch.Metric{
		Dimensions: []cloudwatch.Dimension{{
			Name:  &dimName1,
			Value: &dimValue1,
		},
			{
				Name:  &dimName2,
				Value: &dimValue2,
			}},
		MetricName: &metricName5,
		Namespace:  &namespaceRDS,
	}

	metricsWithStat6 = metricsWithStatistics{
		listMetric6,
		[]string{"Average"},
	}

	listMetricWithDetail6 = listMetricWithDetail{
		metricsWithStats:    []metricsWithStatistics{metricsWithStat6},
		resourceTypeFilters: []string{resourceTypeRDS},
	}

	listMetric7 = cloudwatch.Metric{
		MetricName: &metricName1,
		Namespace:  &namespace,
	}

	namespaceWithDetailS3 = namespaceWithDetail{
		namespace: "AWS/S3",
		statistic: defaultStatistics,
	}
	namespaceWithDetailLambda = namespaceWithDetail{
		namespace: "AWS/Lambda",
		statistic: defaultStatistics,
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
	cases := []struct {
		title                         string
		cloudwatchMetricsConfig       []Config
		expectedlistMetricDetailTotal listMetricWithDetail
		expectednamespaceDetailTotal  []namespaceWithDetail
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
							Value: instanceID1,
						},
					},
					ResourceTypeFilter: resourceTypeEC2,
					Statistic:          []string{"Average"},
				},
			},
			listMetricWithDetail1,
			nil,
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
							Value: instanceID1,
						},
					},
					ResourceTypeFilter: resourceTypeEC2,
					Statistic:          []string{"Average"},
				},
				{
					Namespace: "AWS/S3",
				},
			},
			listMetricWithDetail1,
			[]namespaceWithDetail{namespaceWithDetailS3},
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
							Value: instanceID1,
						},
					},
					Statistic:          []string{"Average"},
					ResourceTypeFilter: resourceTypeEC2,
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
					ResourceTypeFilter: resourceTypeRDS,
				},
			},
			listMetricWithDetail{
				metricsWithStats:    []metricsWithStatistics{metricsWithStat1, metricsWithStat6},
				resourceTypeFilters: []string{resourceTypeEC2, resourceTypeRDS},
			},
			[]namespaceWithDetail{namespaceWithDetailLambda},
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
			listMetricWithDetail{},
			[]namespaceWithDetail{
				{
					namespace:          "AWS/EC2",
					resourceTypeFilter: resourceTypeEC2,
					metricNames:        []string{"CPUUtilization"},
					statistic:          defaultStatistics,
				},
				{
					namespace:          "AWS/S3",
					resourceTypeFilter: "s3",
					statistic:          defaultStatistics,
				},
			},
		},
		{
			"test EBS namespace",
			[]Config{
				{
					Namespace:          "AWS/EBS",
					ResourceTypeFilter: "ec2",
				},
			},
			listMetricWithDetail{},
			[]namespaceWithDetail{{
				namespace:          "AWS/EBS",
				resourceTypeFilter: "ec2",
				statistic:          defaultStatistics,
			}},
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
			listMetricWithDetail{},
			[]namespaceWithDetail{
				{
					namespace:          "AWS/EC2",
					resourceTypeFilter: resourceTypeEC2,
					metricNames:        []string{"CPUUtilization", "StatusCheckFailed"},
					statistic:          []string{"Average", "Maximum"},
				},
			},
		},
		{
			"test AWS/Kafka MemoryUsed",
			[]Config{
				{
					Namespace:  "AWS/Kafka",
					MetricName: []string{"MemoryUsed"},
				},
			},
			listMetricWithDetail{},
			[]namespaceWithDetail{
				{
					namespace:          "AWS/Kafka",
					resourceTypeFilter: "",
					metricNames:        []string{"MemoryUsed"},
					statistic:          defaultStatistics,
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			m.CloudwatchConfigs = c.cloudwatchMetricsConfig
			listMetricDetailTotal, namespaceDetailTotal := m.readCloudwatchConfig()
			assert.Equal(t, c.expectedlistMetricDetailTotal, listMetricDetailTotal)
			assert.Equal(t, c.expectednamespaceDetailTotal, namespaceDetailTotal)
		})
	}
}

func TestGenerateFieldName(t *testing.T) {
	cases := []struct {
		title             string
		label             []string
		expectedFieldName string
	}{
		{
			"test Average",
			[]string{"CPUUtilization", "AWS/EC2", "Average", "InstanceId", "i-1"},
			"aws.metrics.CPUUtilization.avg",
		},
		{
			"test Maximum",
			[]string{"CPUUtilization", "AWS/EC2", "Maximum", "InstanceId", "i-1"},
			"aws.metrics.CPUUtilization.max",
		},
		{
			"test Minimum",
			[]string{"CPUUtilization", "AWS/EC2", "Minimum", "InstanceId", "i-1"},
			"aws.metrics.CPUUtilization.min",
		},
		{
			"test Sum",
			[]string{"CPUUtilization", "AWS/EC2", "Sum", "InstanceId", "i-1"},
			"aws.metrics.CPUUtilization.sum",
		},
		{
			"test SampleCount",
			[]string{"CPUUtilization", "AWS/EC2", "SampleCount", "InstanceId", "i-1"},
			"aws.metrics.CPUUtilization.count",
		},
		{
			"test extended statistic",
			[]string{"CPUUtilization", "AWS/EC2", "p10", "InstanceId", "i-1"},
			"aws.metrics.CPUUtilization.p10",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			fieldName := generateFieldName(c.label)
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
