// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/cloudwatchiface"
	"github.com/stretchr/testify/assert"
)

// MockCloudwatchClient struct is used for unit tests.
type MockCloudWatchClient struct {
	cloudwatchiface.CloudWatchAPI
}

var (
	metricName = "CPUUtilization"
	namespace  = "AWS/EC2"
	dimName    = "InstanceId"
	dimValue   = "i-123"
)

func (m *MockCloudWatchClient) ListMetricsRequest(input *cloudwatch.ListMetricsInput) cloudwatch.ListMetricsRequest {
	dim := cloudwatch.Dimension{
		Name:  &dimName,
		Value: &dimValue,
	}
	return cloudwatch.ListMetricsRequest{
		Request: &awssdk.Request{
			Data: &cloudwatch.ListMetricsOutput{
				Metrics: []cloudwatch.Metric{
					{
						MetricName: &metricName,
						Namespace:  &namespace,
						Dimensions: []cloudwatch.Dimension{dim},
					},
				},
			},
		},
	}
}

func TestGetListMetricsOutput(t *testing.T) {
	svcCloudwatch := &MockCloudWatchClient{}
	listMetricsOutput, err := GetListMetricsOutput("AWS/EC2", "us-west-1", svcCloudwatch)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(listMetricsOutput))
	assert.Equal(t, namespace, *listMetricsOutput[0].Namespace)
	assert.Equal(t, metricName, *listMetricsOutput[0].MetricName)
	assert.Equal(t, 1, len(listMetricsOutput[0].Dimensions))
	assert.Equal(t, dimName, *listMetricsOutput[0].Dimensions[0].Name)
	assert.Equal(t, dimValue, *listMetricsOutput[0].Dimensions[0].Value)
}
