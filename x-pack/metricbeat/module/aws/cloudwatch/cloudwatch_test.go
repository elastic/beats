// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package cloudwatch

import (
	"testing"

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

	listMetricWithDetail1 = listMetricWithDetail{
		listMetric1,
		resourceTypeEC2,
		[]string{"Average"},
	}

	listMetric2 = cloudwatch.Metric{
		Dimensions: []cloudwatch.Dimension{{
			Name:  &dimName,
			Value: &instanceID1,
		}},
		MetricName: &metricName2,
		Namespace:  &namespace,
	}

	listMetricWithDetail2 = listMetricWithDetail{
		listMetric2,
		resourceTypeEC2,
		[]string{"Maximum"},
	}

	listMetric3 = cloudwatch.Metric{
		Dimensions: []cloudwatch.Dimension{{
			Name:  &dimName,
			Value: &instanceID2,
		}},
		MetricName: &metricName3,
		Namespace:  &namespace,
	}

	listMetricWithDetail3 = listMetricWithDetail{
		listMetric3,
		resourceTypeEC2,
		[]string{"Minimum"},
	}

	listMetric4 = cloudwatch.Metric{
		Dimensions: []cloudwatch.Dimension{{
			Name:  &dimName,
			Value: &instanceID2,
		}},
		MetricName: &metricName4,
		Namespace:  &namespace,
	}

	listMetricWithDetail4 = listMetricWithDetail{
		listMetric4,
		resourceTypeEC2,
		[]string{"Sum"},
	}

	listMetric5 = cloudwatch.Metric{
		MetricName: &metricName1,
		Namespace:  &namespace,
	}

	listMetricWithDetail5 = listMetricWithDetail{
		listMetric5,
		resourceTypeRDS,
		[]string{"SampleCount"},
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

	listMetricWithDetail6 = listMetricWithDetail{
		listMetric6,
		resourceTypeRDS,
		[]string{"Average"},
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
)

func TestGetIdentifiers(t *testing.T) {
	listMetricDetailTotal := []listMetricWithDetail{listMetricWithDetail1, listMetricWithDetail2, listMetricWithDetail3, listMetricWithDetail4}
	identifiers := getIdentifiers(listMetricDetailTotal)
	assert.Equal(t, []string{instanceID1, instanceID2}, identifiers["InstanceId"])
}

func TestConstructLabel(t *testing.T) {
	cases := []struct {
		listMetricDetail listMetricWithDetail
		statistic        string
		expectedLabel    string
	}{
		{
			listMetricWithDetail1,
			"Average",
			"CPUUtilization AWS/EC2 InstanceId i-1 Average",
		},
		{
			listMetricWithDetail2,
			"Maximum",
			"StatusCheckFailed AWS/EC2 InstanceId i-1 Maximum",
		},
		{
			listMetricWithDetail3,
			"Minimum",
			"StatusCheckFailed_System AWS/EC2 InstanceId i-2 Minimum",
		},
		{
			listMetricWithDetail4,
			"Sum",
			"StatusCheckFailed_Instance AWS/EC2 InstanceId i-2 Sum",
		},
		{
			listMetricWithDetail5,
			"SampleCount",
			"CPUUtilization AWS/EC2 SampleCount",
		},
	}

	for _, c := range cases {
		label := constructLabel(c.listMetricDetail, c.statistic)
		assert.Equal(t, c.expectedLabel, label)
	}
}

func TestReadCloudwatchConfig(t *testing.T) {
	cases := []struct {
		title                         string
		cloudwatchMetricsConfig       []Config
		expectedlistMetricDetailTotal []listMetricWithDetail
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
			[]listMetricWithDetail{listMetricWithDetail1},
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
			[]listMetricWithDetail{listMetricWithDetail1},
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
			[]listMetricWithDetail{listMetricWithDetail1, listMetricWithDetail6},
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
			[]listMetricWithDetail{
				{
					cloudwatch.Metric{
						MetricName: &metricName1,
						Namespace:  &namespace,
					},
					resourceTypeEC2,
					defaultStatistics,
				},
			},
			[]namespaceWithDetail{{
				namespace:          "AWS/S3",
				resourceTypeFilter: "s3",
				statistic:          defaultStatistics,
			}},
		},
		{
			"test EBS namespace",
			[]Config{
				{
					Namespace:          "AWS/EBS",
					ResourceTypeFilter: "ec2",
				},
			},
			nil,
			[]namespaceWithDetail{{
				namespace:          "AWS/EBS",
				resourceTypeFilter: "ec2",
				statistic:          defaultStatistics,
			}},
		},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			listMetricDetailTotal, namespaceDetailTotal := readCloudwatchConfig(c.cloudwatchMetricsConfig)
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
			[]string{"CPUUtilization", "AWS/EC2", "InstanceId", "i-1", "Average"},
			"metrics.CPUUtilization.avg",
		},
		{
			"test Maximum",
			[]string{"CPUUtilization", "AWS/EC2", "InstanceId", "i-1", "Maximum"},
			"metrics.CPUUtilization.max",
		},
		{
			"test Minimum",
			[]string{"CPUUtilization", "AWS/EC2", "InstanceId", "i-1", "Minimum"},
			"metrics.CPUUtilization.min",
		},
		{
			"test Sum",
			[]string{"CPUUtilization", "AWS/EC2", "InstanceId", "i-1", "Sum"},
			"metrics.CPUUtilization.sum",
		},
		{
			"test SampleCount",
			[]string{"CPUUtilization", "AWS/EC2", "InstanceId", "i-1", "SampleCount"},
			"metrics.CPUUtilization.count",
		},
		{
			"test extended statistic",
			[]string{"CPUUtilization", "AWS/EC2", "InstanceId", "i-1", "p10"},
			"metrics.CPUUtilization.p10",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			fieldName := generateFieldName(c.label)
			assert.Equal(t, c.expectedFieldName, fieldName)
		})
	}
}
