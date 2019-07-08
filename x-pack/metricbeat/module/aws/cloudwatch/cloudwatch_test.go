// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package cloudwatch

import (
	"testing"
	"time"

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

	listMetric1 = cloudwatch.Metric{
		Dimensions: []cloudwatch.Dimension{{
			Name:  &dimName,
			Value: &instanceID1,
		}},
		MetricName: &metricName1,
		Namespace:  &namespace,
	}
	statisticMethod1     = []string{"Average"}
	metricWithStatistic1 = metricStatistic{
		cloudwatchMetric: listMetric1,
		statistic:        statisticMethod1,
	}

	listMetric2 = cloudwatch.Metric{
		Dimensions: []cloudwatch.Dimension{{
			Name:  &dimName,
			Value: &instanceID1,
		}},
		MetricName: &metricName2,
		Namespace:  &namespace,
	}
	statisticMethod2     = []string{"Sum"}
	metricWithStatistic2 = metricStatistic{
		cloudwatchMetric: listMetric2,
		statistic:        statisticMethod2,
	}

	listMetric3 = cloudwatch.Metric{
		Dimensions: []cloudwatch.Dimension{{
			Name:  &dimName,
			Value: &instanceID2,
		}},
		MetricName: &metricName3,
		Namespace:  &namespace,
	}
	statisticMethod3     = []string{"Maximum"}
	metricWithStatistic3 = metricStatistic{
		cloudwatchMetric: listMetric3,
		statistic:        statisticMethod3,
	}

	listMetric4 = cloudwatch.Metric{
		Dimensions: []cloudwatch.Dimension{{
			Name:  &dimName,
			Value: &instanceID2,
		}},
		MetricName: &metricName4,
		Namespace:  &namespace,
	}
	statisticMethod4     = []string{"Minimum"}
	metricWithStatistic4 = metricStatistic{
		cloudwatchMetric: listMetric4,
		statistic:        statisticMethod4,
	}

	listMetric5 = cloudwatch.Metric{
		MetricName: &metricName1,
		Namespace:  &namespace,
	}
	statisticMethod5     = []string{"Average", "Sum"}
	metricWithStatistic5 = metricStatistic{
		cloudwatchMetric: listMetric5,
		statistic:        statisticMethod5,
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
	statisticMethod6     = []string{"Maximum", "Minimum"}
	metricWithStatistic6 = metricStatistic{
		cloudwatchMetric: listMetric6,
		statistic:        statisticMethod6,
	}

	listMetric7 = cloudwatch.Metric{
		MetricName: &metricName1,
		Namespace:  &namespace,
	}
)

func TestGetIdentifiers(t *testing.T) {
	cloudwatchMetricWithStatistic := []metricStatistic{
		metricWithStatistic1,
		metricWithStatistic2,
		metricWithStatistic3,
		metricWithStatistic4,
	}
	identifiers := getIdentifiers(cloudwatchMetricWithStatistic)
	assert.Equal(t, []string{instanceID1, instanceID2}, identifiers["InstanceId"])
}

func TestConstructLabel(t *testing.T) {
	cases := []struct {
		listMetric      cloudwatch.Metric
		statisticMethod []string
		expectedLabel   string
	}{
		{
			listMetric1,
			statisticMethod1,
			"CPUUtilization AWS/EC2 InstanceId i-1 Average",
		},
		{
			listMetric2,
			statisticMethod2,
			"StatusCheckFailed AWS/EC2 InstanceId i-1 Sum",
		},
		{
			listMetric3,
			statisticMethod3,
			"StatusCheckFailed_System AWS/EC2 InstanceId i-2 Maximum",
		},
		{
			listMetric4,
			statisticMethod4,
			"StatusCheckFailed_Instance AWS/EC2 InstanceId i-2 Minimum",
		},
	}

	for _, c := range cases {
		label := constructLabel(c.listMetric, c.statisticMethod[0])
		assert.Equal(t, c.expectedLabel, label)
	}
}

func TestConvertConfigToMetricStatistics(t *testing.T) {
	cases := []struct {
		title                   string
		cloudwatchMetricsConfig Config
		expectedMetricStatistic []metricStatistic
	}{
		{
			"test with a specific metric with statistic",
			Config{
				Namespace: "AWS/EC2",
				Metrics: []metric{
					{
						Names: []string{"CPUUtilization"},
						Dimensions: []dimension{
							{
								Name:  "InstanceId",
								Value: instanceID1,
							},
						},
						Statistics: statisticMethod1,
					},
				},
			},
			[]metricStatistic{metricWithStatistic1},
		},
		{
			"test with a specific metric without statistic",
			Config{
				Namespace: "AWS/EC2",
				Metrics: []metric{
					{
						Names: []string{"CPUUtilization"},
						Dimensions: []dimension{
							{
								Name:  "InstanceId",
								Value: instanceID1,
							},
						},
					},
				},
			},
			[]metricStatistic{
				{
					cloudwatchMetric: listMetric1,
					statistic:        []string{"Average", "Maximum", "Minimum", "Sum", "SampleCount"}},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			metricWithStatisticTotal := convertConfigToMetricStatistics(c.cloudwatchMetricsConfig)
			assert.Equal(t, 1, len(metricWithStatisticTotal))
			assert.Equal(t, c.expectedMetricStatistic, metricWithStatisticTotal)
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

func TestCreateMetricDataQueries(t *testing.T) {
	periodInSec := int64(60)
	cases := []struct {
		title               string
		metricWithStatistic []metricStatistic
		expectedQueries     []cloudwatch.MetricDataQuery
	}{
		{
			"test with single metric and statistic method",
			[]metricStatistic{
				{[]string{"Average"},
					listMetric1},
			},
			[]cloudwatch.MetricDataQuery{
				{
					Id:    awssdk.String("cw0stats0"),
					Label: awssdk.String("CPUUtilization AWS/EC2 InstanceId i-1 Average"),
					MetricStat: &cloudwatch.MetricStat{
						Period: &periodInSec,
						Stat:   awssdk.String("Average"),
						Metric: &listMetric1,
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			metricDataQueries := createMetricDataQueries(c.metricWithStatistic, time.Duration(periodInSec)*time.Second)
			assert.Equal(t, 1, len(metricDataQueries))
			assert.Equal(t, c.expectedQueries, metricDataQueries)
		})
	}
}
