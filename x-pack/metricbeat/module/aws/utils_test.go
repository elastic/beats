// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/stretchr/testify/assert"
)

func TestGetStartTimeEndTime(t *testing.T) {
	_, _, err := GetStartTimeEndTime("-20m")
	assert.NoError(t, err)
}

func TestConstructMetricQueries(t *testing.T) {
	name1 := "StorageType"
	value1 := "AllStorageTypes"
	dim1 := cloudwatch.Dimension{
		Name:  &name1,
		Value: &value1,
	}

	name2 := "BucketName"
	value2 := "test-s3-bucket"
	dim2 := cloudwatch.Dimension{
		Name:  &name2,
		Value: &value2,
	}
	metricName := "NumberOfObjects"
	namespace := "AWS/S3"
	listMetric := cloudwatch.Metric{
		Dimensions: []cloudwatch.Dimension{dim1, dim2},
		MetricName: &metricName,
		Namespace:  &namespace,
	}

	period := int64(300)
	listMetricsOutput := []cloudwatch.Metric{listMetric}
	whiteList := []string{"NumberOfObjects"}
	metricDataQuery1 := ConstructMetricQueries(listMetricsOutput, period, whiteList, nil)
	assert.Equal(t, 1, len(metricDataQuery1))
	assert.Equal(t, "test-s3-bucket AllStorageTypes  NumberOfObjects", *metricDataQuery1[0].Label)
	assert.Equal(t, "Average", *metricDataQuery1[0].MetricStat.Stat)
	assert.Equal(t, period, *metricDataQuery1[0].MetricStat.Period)
	assert.Equal(t, "NumberOfObjects", *metricDataQuery1[0].MetricStat.Metric.MetricName)
	assert.Equal(t, "AWS/S3", *metricDataQuery1[0].MetricStat.Metric.Namespace)

	blackList := []string{"NumberOfObjects"}
	metricDataQuery2 := ConstructMetricQueries(listMetricsOutput, period, nil, blackList)
	assert.Equal(t, 0, len(metricDataQuery2))
}
