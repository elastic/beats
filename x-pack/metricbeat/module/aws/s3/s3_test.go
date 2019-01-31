// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package s3

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3iface"
	"github.com/stretchr/testify/assert"
)

// MockS3Client struct is used for unit tests.
type MockS3Client struct {
	s3iface.S3API
}

// MockCloudWatchClient struct is used for unit tests.
type MockCloudWatchClient struct {
	cloudwatchiface.CloudWatchAPI
}

var regionName = "us-west-1"

func TestGetStartTimeEndTime(t *testing.T) {
	duration := "300s"
	_, _, err := getStartTimeEndTime(duration)
	assert.NoError(t, err)

	duration = "300ss"
	_, _, err = getStartTimeEndTime(duration)
	assert.Error(t, err)
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

	listMetricsOutput := []cloudwatch.Metric{listMetric}
	metricDataQuery := constructMetricQueries(listMetricsOutput)
	assert.Equal(t, 1, len(metricDataQuery))
	assert.Equal(t, "test-s3-bucket AllStorageTypes NumberOfObjects", *metricDataQuery[0].Label)
	assert.Equal(t, "Average", *metricDataQuery[0].MetricStat.Stat)
	assert.Equal(t, int64(86400), *metricDataQuery[0].MetricStat.Period)
	assert.Equal(t, "NumberOfObjects", *metricDataQuery[0].MetricStat.Metric.MetricName)
	assert.Equal(t, "AWS/S3", *metricDataQuery[0].MetricStat.Metric.Namespace)
}
