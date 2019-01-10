// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package ec2

import (
	"fmt"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

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

func (m *MockEC2Client) DescribeInstancesRequest(input *ec2.DescribeInstancesInput) ec2.DescribeInstancesRequest {
	monitoringState := ec2.Monitoring{
		State: ec2.MonitoringState(ec2.MonitoringStateDisabled),
	}
	instance1 := ec2.Instance{
		InstanceId:   awssdk.String("i-123"),
		InstanceType: ec2.InstanceTypeT2Medium,
		Placement:    &ec2.Placement{AvailabilityZone: awssdk.String("us-west-1a")},
		ImageId:      awssdk.String("image-123"),
		Monitoring:   &monitoringState,
	}
	instance2 := ec2.Instance{
		InstanceId:   awssdk.String("i-456"),
		InstanceType: ec2.InstanceTypeT2Micro,
		Placement:    &ec2.Placement{AvailabilityZone: awssdk.String("us-west-1b")},
		ImageId:      awssdk.String("image-456"),
		Monitoring:   &monitoringState,
	}
	return ec2.DescribeInstancesRequest{
		Request: &awssdk.Request{
			Data: &ec2.DescribeInstancesOutput{
				Reservations: []ec2.RunInstancesOutput{
					{Instances: []ec2.Instance{instance1, instance2}},
				},
			},
		},
	}
}

func (m *MockCloudWatchClient) GetMetricDataRequest(input *cloudwatch.GetMetricDataInput) cloudwatch.GetMetricDataRequest {
	id := "cpu1"
	label := "CPUUtilization"
	value := 0.25
	return cloudwatch.GetMetricDataRequest{
		Request: &awssdk.Request{
			Data: &cloudwatch.GetMetricDataOutput{
				MetricDataResults: []cloudwatch.MetricDataResult{
					cloudwatch.MetricDataResult{
						Id:     &id,
						Label:  &label,
						Values: []float64{value},
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

func TestGetInstanceIDs(t *testing.T) {
	mockSvc := &MockEC2Client{}
	instanceIDs, instancesOutputs, err := getInstancesPerRegion(mockSvc)
	if err != nil {
		fmt.Println("failed getInstancesPerRegion: ", err)
		t.FailNow()
	}

	assert.Equal(t, 2, len(instanceIDs))
	assert.Equal(t, 2, len(instancesOutputs))

	assert.Equal(t, "i-123", instanceIDs[0])
	assert.Equal(t, ec2.InstanceType("t2.medium"), instancesOutputs["i-123"].InstanceType)
	assert.Equal(t, awssdk.String("image-123"), instancesOutputs["i-123"].ImageId)
	assert.Equal(t, awssdk.String("us-west-1a"), instancesOutputs["i-123"].Placement.AvailabilityZone)

	assert.Equal(t, "i-456", instanceIDs[1])
	assert.Equal(t, ec2.InstanceType("t2.micro"), instancesOutputs["i-456"].InstanceType)
	assert.Equal(t, awssdk.String("us-west-1b"), instancesOutputs["i-456"].Placement.AvailabilityZone)
}

func TestGetMetricDataPerRegion(t *testing.T) {
	mockSvc := &MockCloudWatchClient{}
	getMetricDataOutput, err := getMetricDataPerRegion("-10m", 300, "i-123", nil, mockSvc)
	if err != nil {
		fmt.Println("failed getMetricDataPerRegion: ", err)
		t.FailNow()
	}
	assert.Equal(t, 1, len(getMetricDataOutput.MetricDataResults))
	assert.Equal(t, "cpu1", *getMetricDataOutput.MetricDataResults[0].Id)
	assert.Equal(t, "CPUUtilization", *getMetricDataOutput.MetricDataResults[0].Label)
	assert.Equal(t, 0.25, getMetricDataOutput.MetricDataResults[0].Values[0])
}

func TestConvertPeriodToDurationWithDetailedMonitoring(t *testing.T) {
	period1 := "300s"
	duration1, periodSec1 := convertPeriodToDuration(period1, "enabled")
	assert.Equal(t, "-600s", duration1)
	assert.Equal(t, 300, periodSec1)

	period2 := "30ss"
	duration2, periodSec2 := convertPeriodToDuration(period2, "enabled")
	assert.Equal(t, "-120s", duration2)
	assert.Equal(t, 60, periodSec2)

	period3 := "10m"
	duration3, periodSec3 := convertPeriodToDuration(period3, "enabled")
	assert.Equal(t, "-20m", duration3)
	assert.Equal(t, 600, periodSec3)

	period4 := "30s"
	duration4, periodSec4 := convertPeriodToDuration(period4, "enabled")
	assert.Equal(t, "-120s", duration4)
	assert.Equal(t, 60, periodSec4)

	period5 := "60s"
	duration5, periodSec5 := convertPeriodToDuration(period5, "enabled")
	assert.Equal(t, "-120s", duration5)
	assert.Equal(t, 60, periodSec5)
}

func TestConvertPeriodToDurationWithBasicMonitoring(t *testing.T) {
	period1 := "300s"
	duration1, periodSec1 := convertPeriodToDuration(period1, "disabled")
	assert.Equal(t, "-600s", duration1)
	assert.Equal(t, 300, periodSec1)

	period2 := "30ss"
	duration2, periodSec2 := convertPeriodToDuration(period2, "disabled")
	assert.Equal(t, "-600s", duration2)
	assert.Equal(t, 300, periodSec2)

	period3 := "10m"
	duration3, periodSec3 := convertPeriodToDuration(period3, "disabled")
	assert.Equal(t, "-20m", duration3)
	assert.Equal(t, 600, periodSec3)

	period5 := "60s"
	duration5, periodSec5 := convertPeriodToDuration(period5, "disabled")
	assert.Equal(t, "-600s", duration5)
	assert.Equal(t, 300, periodSec5)
}

func TestMockFetch(t *testing.T) {
	mockCreds := map[string]interface{}{
		"module":            "aws",
		"period":            "300s",
		"metricsets":        []string{"ec2"},
		"access_key_id":     "mock",
		"secret_access_key": "mock",
	}

	awsMetricSet := mbtest.NewReportingMetricSetV2(t, mockCreds)
	events, errs := mbtest.ReportingFetchV2(awsMetricSet)
	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}
	t.Logf("Module: %s Metricset: %s", awsMetricSet.Module().Name(), awsMetricSet.Name())

	assert.Equal(t, 2, len(events))
	for _, event := range events {
		// MetricSetField
		cpuTotalPct, err := event.MetricSetFields.GetValue("cpu.total.pct")
		assert.NoError(t, err)
		assert.Equal(t, 0.25, cpuTotalPct)
	}
}
