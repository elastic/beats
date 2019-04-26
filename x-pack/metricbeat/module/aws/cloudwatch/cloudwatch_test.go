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
	listMetricsOutput := []cloudwatch.Metric{listMetric1, listMetric2, listMetric3, listMetric4}
	identifiers := getIdentifiers(listMetricsOutput)
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
		cloudwatchMetricsConfig []Config
		expectedListMetric      []cloudwatch.Metric
		expectedNamespace       []string
	}{
		{
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
			[]cloudwatch.Metric{listMetric1},
			nil,
		},
		{
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
					Namespace: "AWS/EBS",
				},
			},
			[]cloudwatch.Metric{listMetric1},
			[]string{"AWS/EBS"},
		},
		{
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
					Namespace: "AWS/EBS",
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
			[]cloudwatch.Metric{listMetric1, listMetric6},
			[]string{"AWS/EBS"},
		},
		{
			[]Config{
				{
					Namespace:  "AWS/EC2",
					MetricName: "CPUUtilization",
				},
				{
					Namespace: "AWS/EBS",
				},
			},
			[]cloudwatch.Metric{listMetric7},
			[]string{"AWS/EBS"},
		},
	}
	for _, c := range cases {
		listMetrics, namespaces := readCloudwatchConfig(c.cloudwatchMetricsConfig)
		assert.Equal(t, c.expectedListMetric, listMetrics)
		assert.Equal(t, c.expectedNamespace, namespaces)
	}
}
