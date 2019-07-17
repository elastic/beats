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
	instanceID1  = "i-1"
	instanceID2  = "i-2"
	namespace    = "AWS/EC2"
	dimName      = "InstanceId"
	metricName1  = "CPUUtilization"
	metricName2  = "StatusCheckFailed"
	metricName3  = "StatusCheckFailed_System"
	metricName4  = "StatusCheckFailed_Instance"
	metricName5  = "CommitThroughput"
	namespaceRDS = "AWS/RDS"
	listMetric1  = cloudwatch.Metric{
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

	listMetric7 = cloudwatch.Metric{
		MetricName: &metricName1,
		Namespace:  &namespace,
	}
)

func TestGetIdentifiers(t *testing.T) {
	listMetricsTotal := []cloudwatch.Metric{
		listMetric1, listMetric2, listMetric3, listMetric4,
	}

	identifiers := getIdentifiers(listMetricsTotal)
	assert.Equal(t, []string{instanceID1, instanceID2}, identifiers["InstanceId"])
}

func TestConstructLabel(t *testing.T) {
	cases := []struct {
		listMetric    cloudwatch.Metric
		expectedLabel string
	}{
		{
			listMetric1,
			"CPUUtilization AWS/EC2 InstanceId i-1",
		},
		{
			listMetric2,
			"StatusCheckFailed AWS/EC2 InstanceId i-1",
		},
		{
			listMetric3,
			"StatusCheckFailed_System AWS/EC2 InstanceId i-2",
		},
		{
			listMetric4,
			"StatusCheckFailed_Instance AWS/EC2 InstanceId i-2",
		},
		{
			listMetric5,
			"CPUUtilization AWS/EC2",
		},
	}

	for _, c := range cases {
		label := constructLabel(c.listMetric)
		assert.Equal(t, c.expectedLabel, label)
	}
}

func TestReadCloudwatchConfig(t *testing.T) {
	cases := []struct {
		title                    string
		cloudwatchMetricsConfig  []Config
		expectedListMetricDetail listMetricWithDetail
		expectedNamespaceDetail  []namespaceWithDetail
	}{
		{
			"test with a specific metric",
			[]Config{
				{
					Namespace:  "AWS/EC2",
					MetricName: "CPUUtilization",
					Dimensions: []Dimension{
						{
							Name:  "InstanceId",
							Value: instanceID1,
						},
					},
				},
			},
			listMetricWithDetail{
				cloudwatchMetrics:   []cloudwatch.Metric{listMetric1},
				resourceTypeFilters: nil,
			},
			nil,
		},
		{
			"test with a namespace",
			[]Config{
				{
					Namespace: "AWS/EC2",
				},
			},
			listMetricWithDetail{},
			[]namespaceWithDetail{{namespace: "AWS/EC2", resourceTypeFilter: ""}},
		},
		{
			"test with a specific metric and a namespace",
			[]Config{
				{
					Namespace:  "AWS/EC2",
					MetricName: "CPUUtilization",
					Dimensions: []Dimension{
						{
							Name:  "InstanceId",
							Value: instanceID1,
						},
					},
				},
				{
					Namespace: "AWS/S3",
				},
			},
			listMetricWithDetail{
				cloudwatchMetrics:   []cloudwatch.Metric{listMetric1},
				resourceTypeFilters: nil,
			},
			[]namespaceWithDetail{{namespace: "AWS/S3", resourceTypeFilter: ""}},
		},
		{
			"test with two specific metrics and a namespace",
			[]Config{
				{
					Namespace:  "AWS/EC2",
					MetricName: "CPUUtilization",
					Dimensions: []Dimension{
						{
							Name:  "InstanceId",
							Value: instanceID1,
						},
					},
					ResourceTypeFilter: "ec2:instance",
				},
				{
					Namespace: "AWS/Lambda",
				},
				{
					Namespace:  "AWS/RDS",
					MetricName: "CommitThroughput",
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
				},
			},
			listMetricWithDetail{
				cloudwatchMetrics:   []cloudwatch.Metric{listMetric1, listMetric6},
				resourceTypeFilters: []string{"ec2:instance"},
			},
			[]namespaceWithDetail{{namespace: "AWS/Lambda", resourceTypeFilter: ""}},
		},
		{
			"Test a specific metric (only with metric name) and a namespace",
			[]Config{
				{
					Namespace:          "AWS/EC2",
					MetricName:         "CPUUtilization",
					ResourceTypeFilter: "ec2:instance",
				},
				{
					Namespace:          "AWS/S3",
					ResourceTypeFilter: "s3",
				},
			},
			listMetricWithDetail{},
			[]namespaceWithDetail{
				{namespace: "AWS/EC2", resourceTypeFilter: "ec2:instance", metricName: "CPUUtilization"},
				{namespace: "AWS/S3", resourceTypeFilter: "s3"},
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
			[]namespaceWithDetail{{namespace: "AWS/EBS", resourceTypeFilter: "ec2"}},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			m := &MetricSet{CloudwatchConfigs: c.cloudwatchMetricsConfig}
			listMetricDetailTotal, namespaceDetailTotal := m.readCloudwatchConfig()
			assert.Equal(t, len(c.expectedListMetricDetail.resourceTypeFilters), len(listMetricDetailTotal.resourceTypeFilters))
			assert.Equal(t, len(c.expectedListMetricDetail.cloudwatchMetrics), len(listMetricDetailTotal.cloudwatchMetrics))
			assert.Equal(t, len(c.expectedNamespaceDetail), len(namespaceDetailTotal))
			assert.Equal(t, c.expectedListMetricDetail, listMetricDetailTotal)
			assert.Equal(t, c.expectedNamespaceDetail, namespaceDetailTotal)
		})
	}
}

func TestCompareAWSDimensions(t *testing.T) {
	cases := []struct {
		title                    string
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
