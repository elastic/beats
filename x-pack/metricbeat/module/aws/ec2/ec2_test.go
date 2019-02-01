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
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws"

	"github.com/elastic/beats/libbeat/common"
)

// MockEC2Client struct is used for unit tests.
type MockEC2Client struct {
	ec2iface.EC2API
}

// MockCloudWatchClient struct is used for unit tests.
type MockCloudWatchClient struct {
	cloudwatchiface.CloudWatchAPI
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

func (m *MockEC2Client) DescribeInstancesRequest(input *ec2.DescribeInstancesInput) ec2.DescribeInstancesRequest {
	runningCode := int64(16)
	coreCount := int64(1)
	threadsPerCore := int64(1)
	publicDNSName := "ec2-1-2-3-4.us-west-1.compute.amazonaws.com"
	publicIP := "1.2.3.4"
	privateDNSName := "ip-5-6-7-8.us-west-1.compute.internal"
	privateIP := "5.6.7.8"

	instance := ec2.Instance{
		InstanceId:   awssdk.String("i-123"),
		InstanceType: ec2.InstanceTypeT2Medium,
		Placement: &ec2.Placement{
			AvailabilityZone: awssdk.String("us-west-1a"),
		},
		ImageId: awssdk.String("image-123"),
		State: &ec2.InstanceState{
			Name: ec2.InstanceStateNameRunning,
			Code: &runningCode,
		},
		Monitoring: &ec2.Monitoring{
			State: ec2.MonitoringStateDisabled,
		},
		CpuOptions: &ec2.CpuOptions{
			CoreCount:      &coreCount,
			ThreadsPerCore: &threadsPerCore,
		},
		PublicDnsName:    &publicDNSName,
		PublicIpAddress:  &publicIP,
		PrivateDnsName:   &privateDNSName,
		PrivateIpAddress: &privateIP,
	}
	return ec2.DescribeInstancesRequest{
		Request: &awssdk.Request{
			Data: &ec2.DescribeInstancesOutput{
				Reservations: []ec2.RunInstancesOutput{
					{Instances: []ec2.Instance{instance}},
				},
			},
		},
	}
}

func (m *MockCloudWatchClient) GetMetricDataRequest(input *cloudwatch.GetMetricDataInput) cloudwatch.GetMetricDataRequest {
	id1 := "cpu1"
	label1 := "CPUUtilization"
	value1 := 0.25

	id2 := "status1"
	label2 := "StatusCheckFailed"
	value2 := 0.0

	id3 := "status2"
	label3 := "StatusCheckFailed_System"
	value3 := 0.0

	id4 := "status3"
	label4 := "StatusCheckFailed_Instance"
	value4 := 0.0

	return cloudwatch.GetMetricDataRequest{
		Request: &awssdk.Request{
			Data: &cloudwatch.GetMetricDataOutput{
				MetricDataResults: []cloudwatch.MetricDataResult{
					{
						Id:     &id1,
						Label:  &label1,
						Values: []float64{value1},
					},
					{
						Id:     &id2,
						Label:  &label2,
						Values: []float64{value2},
					},
					{
						Id:     &id3,
						Label:  &label3,
						Values: []float64{value3},
					},
					{
						Id:     &id4,
						Label:  &label4,
						Values: []float64{value4},
					},
				},
			},
		},
	}
}

func TestGetInstanceIDs(t *testing.T) {
	mockSvc := &MockEC2Client{}
	instanceIDs, instancesOutputs, err := getInstancesPerRegion(mockSvc)
	if err != nil {
		fmt.Println("failed getInstancesPerRegion: ", err)
		t.FailNow()
	}

	assert.Equal(t, 1, len(instanceIDs))
	assert.Equal(t, 1, len(instancesOutputs))

	assert.Equal(t, "i-123", instanceIDs[0])
	assert.Equal(t, ec2.InstanceType("t2.medium"), instancesOutputs["i-123"].InstanceType)
	assert.Equal(t, awssdk.String("image-123"), instancesOutputs["i-123"].ImageId)
	assert.Equal(t, awssdk.String("us-west-1a"), instancesOutputs["i-123"].Placement.AvailabilityZone)
}

func TestGetMetricDataPerRegion(t *testing.T) {
	mockSvc := &MockCloudWatchClient{}
	getMetricDataOutput, err := getMetricDataPerRegion("-10m", 300, "i-123", nil, mockSvc)
	if err != nil {
		fmt.Println("failed getMetricDataPerRegion: ", err)
		t.FailNow()
	}
	assert.Equal(t, 4, len(getMetricDataOutput.MetricDataResults))
	assert.Equal(t, "cpu1", *getMetricDataOutput.MetricDataResults[0].Id)
	assert.Equal(t, "CPUUtilization", *getMetricDataOutput.MetricDataResults[0].Label)
	assert.Equal(t, 0.25, getMetricDataOutput.MetricDataResults[0].Values[0])

	assert.Equal(t, "status1", *getMetricDataOutput.MetricDataResults[1].Id)
	assert.Equal(t, "StatusCheckFailed", *getMetricDataOutput.MetricDataResults[1].Label)
	assert.Equal(t, 0.0, getMetricDataOutput.MetricDataResults[1].Values[0])

	assert.Equal(t, "status2", *getMetricDataOutput.MetricDataResults[2].Id)
	assert.Equal(t, "StatusCheckFailed_System", *getMetricDataOutput.MetricDataResults[2].Label)
	assert.Equal(t, 0.0, getMetricDataOutput.MetricDataResults[2].Values[0])

	assert.Equal(t, "status3", *getMetricDataOutput.MetricDataResults[3].Id)
	assert.Equal(t, "StatusCheckFailed_Instance", *getMetricDataOutput.MetricDataResults[3].Label)
	assert.Equal(t, 0.0, getMetricDataOutput.MetricDataResults[3].Values[0])
}

func TestCreateCloudWatchEvents(t *testing.T) {
	mockModuleConfig := aws.Config{
		Period:        "300s",
		DefaultRegion: regionName,
	}

	expectedEvent := mb.Event{
		RootFields: common.MapStr{
			"service": common.MapStr{"name": "ec2"},
			"cloud": common.MapStr{
				"region":            regionName,
				"provider":          "ec2",
				"instance":          common.MapStr{"id": "i-123"},
				"machine":           common.MapStr{"type": "t2.medium"},
				"availability_zone": "us-west-1a",
			},
		},
		MetricSetFields: common.MapStr{
			"cpu": common.MapStr{
				"total": common.MapStr{"pct": 0.25},
			},
			"instance": common.MapStr{
				"image":            common.MapStr{"id": "image-123"},
				"core":             common.MapStr{"count": int64(1)},
				"threads_per_core": int64(1),
				"state":            common.MapStr{"code": int64(16), "name": "running"},
				"monitoring":       common.MapStr{"state": "disabled"},
				"public": common.MapStr{
					"dns_name": "ec2-1-2-3-4.us-west-1.compute.amazonaws.com",
					"ip":       "1.2.3.4",
				},
				"private": common.MapStr{
					"dns_name": "ip-5-6-7-8.us-west-1.compute.internal",
					"ip":       "5.6.7.8",
				},
			},
		},
	}
	svcEC2Mock := &MockEC2Client{}
	instanceIDs, instancesOutputs, err := getInstancesPerRegion(svcEC2Mock)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(instanceIDs))
	instanceID := instanceIDs[0]
	assert.Equal(t, "i-123", instanceID)

	svcCloudwatchMock := &MockCloudWatchClient{}
	getMetricDataOutput, err := getMetricDataPerRegion("-600s", 300, instanceID, nil, svcCloudwatchMock)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(getMetricDataOutput.MetricDataResults))
	assert.Equal(t, "cpu1", *getMetricDataOutput.MetricDataResults[0].Id)
	assert.Equal(t, "CPUUtilization", *getMetricDataOutput.MetricDataResults[0].Label)
	assert.Equal(t, 0.25, getMetricDataOutput.MetricDataResults[0].Values[0])

	event, info, err := createCloudWatchEvents(getMetricDataOutput, instanceID, instancesOutputs[instanceID], mockModuleConfig.DefaultRegion)
	assert.NoError(t, err)
	assert.Equal(t, "", info)
	assert.Equal(t, expectedEvent.RootFields, event.RootFields)
	assert.Equal(t, expectedEvent.MetricSetFields["cpu"], event.MetricSetFields["cpu"])
	assert.Equal(t, expectedEvent.MetricSetFields["instance"], event.MetricSetFields["instance"])
}
