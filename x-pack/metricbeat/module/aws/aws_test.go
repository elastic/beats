// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package aws

import (
	"fmt"
	"testing"
	"time"

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
	cases := []struct {
		period               time.Duration
		expectedDuration     string
		expectedPeriodNumber int
	}{
		{
			period:               time.Duration(300) * time.Second,
			expectedDuration:     "-10m0s",
			expectedPeriodNumber: 300,
		},
		{
			period:               time.Duration(10) * time.Minute,
			expectedDuration:     "-20m0s",
			expectedPeriodNumber: 600,
		},
		{
			period:               time.Duration(30) * time.Second,
			expectedDuration:     "-1m0s",
			expectedPeriodNumber: 30,
		},
		{
			period:               time.Duration(60) * time.Second,
			expectedDuration:     "-2m0s",
			expectedPeriodNumber: 60,
		},
	}

	for _, c := range cases {
		duration, periodSec := convertPeriodToDuration(c.period)
		assert.Equal(t, c.expectedDuration, duration)
		assert.Equal(t, c.expectedPeriodNumber, periodSec)
	}
}
