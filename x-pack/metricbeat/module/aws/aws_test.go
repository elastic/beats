// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package aws

import (
	"fmt"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
	"github.com/stretchr/testify/assert"
)

// MockEC2Client struct is used for unit tests.
type MockEC2Client struct {
	ec2iface.EC2API
}

var regionName = "us-west-1"

func (m *MockEC2Client) DescribeRegionsRequest(input *ec2.DescribeRegionsInput) ec2.DescribeRegionsRequest {
	return ec2.DescribeRegionsRequest{
		Request: &awssdk.Request{
			Data: &ec2.DescribeRegionsOutput{
				Regions: []ec2.Region{
					{
						RegionName: &regionName,
					},
				},
			},
		},
	}
}

func TestGetRegions(t *testing.T) {
	mockSvc := &MockEC2Client{}
	regionsList, err := getRegions(mockSvc)
	if err != nil {
		fmt.Println("failed getRegions: ", err)
		t.FailNow()
	}
	assert.Equal(t, 1, len(regionsList))
	assert.Equal(t, regionName, regionsList[0])
}

func TestConvertPeriodToDuration(t *testing.T) {
	period1 := "300s"
	duration1, periodSec1, err := convertPeriodToDuration(period1)
	assert.NoError(t, nil, err)
	assert.Equal(t, "-600s", duration1)
	assert.Equal(t, 300, periodSec1)

	period2 := "30ss"
	duration2, periodSec2, err := convertPeriodToDuration(period2)
	assert.Error(t, err)
	assert.Equal(t, "", duration2)
	assert.Equal(t, 0, periodSec2)

	period3 := "10m"
	duration3, periodSec3, err := convertPeriodToDuration(period3)
	assert.NoError(t, nil, err)
	assert.Equal(t, "-20m", duration3)
	assert.Equal(t, 600, periodSec3)

	period4 := "30s"
	duration4, periodSec4, err := convertPeriodToDuration(period4)
	assert.NoError(t, nil, err)
	assert.Equal(t, "-60s", duration4)
	assert.Equal(t, 30, periodSec4)

	period5 := "60s"
	duration5, periodSec5, err := convertPeriodToDuration(period5)
	assert.NoError(t, nil, err)
	assert.Equal(t, "-120s", duration5)
	assert.Equal(t, 60, periodSec5)
}
