// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package ec2

import (
	"net/http"
	"testing"
	"time"

	"github.com/elastic/beats/x-pack/metricbeat/module/aws"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/ec2iface"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

// MockEC2Client struct is used for unit tests.
type MockEC2Client struct {
	ec2iface.ClientAPI
}

var (
	regionName = "us-west-1"
	instanceID = "i-123"
	namespace  = "AWS/EC2"

	id1         = "cpu1"
	metricName1 = "CPUUtilization"
	label1      = instanceID + labelSeparator + metricName1

	id2         = "status1"
	metricName2 = "StatusCheckFailed"
	label2      = instanceID + labelSeparator + metricName2

	id3         = "status2"
	metricName3 = "StatusCheckFailed_System"
	label3      = instanceID + labelSeparator + metricName3

	id4         = "status3"
	metricName4 = "StatusCheckFailed_Instance"
	label4      = instanceID + labelSeparator + metricName4
)

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
		InstanceId:   awssdk.String(instanceID),
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

	httpReq, _ := http.NewRequest("", "", nil)
	return ec2.DescribeInstancesRequest{
		Request: &awssdk.Request{
			Data: &ec2.DescribeInstancesOutput{
				Reservations: []ec2.Reservation{
					{Instances: []ec2.Instance{instance}},
				},
			},
			HTTPRequest: httpReq,
		},
	}
}

func TestGetInstanceIDs(t *testing.T) {
	mockSvc := &MockEC2Client{}
	instanceIDs, instancesOutputs, err := getInstancesPerRegion(mockSvc)
	if err != nil {
		t.FailNow()
	}

	assert.Equal(t, 1, len(instanceIDs))
	assert.Equal(t, 1, len(instancesOutputs))

	assert.Equal(t, instanceID, instanceIDs[0])
	assert.Equal(t, ec2.InstanceType("t2.medium"), instancesOutputs[instanceID].InstanceType)
	assert.Equal(t, awssdk.String("image-123"), instancesOutputs[instanceID].ImageId)
	assert.Equal(t, awssdk.String("us-west-1a"), instancesOutputs[instanceID].Placement.AvailabilityZone)
}

func TestCreateCloudWatchEvents(t *testing.T) {
	expectedEvent := mb.Event{
		RootFields: common.MapStr{
			"cloud": common.MapStr{
				"region":            regionName,
				"provider":          "aws",
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
	assert.Equal(t, instanceID, instanceID)
	timestamp := time.Now()

	getMetricDataOutput := []cloudwatch.MetricDataResult{
		{
			Id:         &id1,
			Label:      &label1,
			Values:     []float64{0.25},
			Timestamps: []time.Time{timestamp},
		},
		{
			Id:         &id2,
			Label:      &label2,
			Values:     []float64{0.0},
			Timestamps: []time.Time{timestamp},
		},
		{
			Id:         &id3,
			Label:      &label3,
			Values:     []float64{0.0},
			Timestamps: []time.Time{timestamp},
		},
		{
			Id:         &id4,
			Label:      &label4,
			Values:     []float64{0.0},
			Timestamps: []time.Time{timestamp},
		},
	}

	metricSet := MetricSet{
		&aws.MetricSet{},
	}
	events, err := metricSet.createCloudWatchEvents(getMetricDataOutput, instancesOutputs, "us-west-1")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
	assert.Equal(t, expectedEvent.RootFields, events[instanceID].RootFields)
	assert.Equal(t, expectedEvent.MetricSetFields["cpu"], events[instanceID].MetricSetFields["cpu"])
	assert.Equal(t, expectedEvent.MetricSetFields["instance"], events[instanceID].MetricSetFields["instance"])
}

func TestConstructMetricQueries(t *testing.T) {
	name := "InstanceId"
	dim := cloudwatch.Dimension{
		Name:  &name,
		Value: &instanceID,
	}

	listMetric := cloudwatch.Metric{
		Dimensions: []cloudwatch.Dimension{dim},
		MetricName: &metricName1,
		Namespace:  &namespace,
	}

	listMetricsOutput := []cloudwatch.Metric{listMetric}
	metricDataQuery := constructMetricQueries(listMetricsOutput, instanceID, 5*time.Minute)
	assert.Equal(t, 1, len(metricDataQuery))
	assert.Equal(t, "i-123|CPUUtilization", *metricDataQuery[0].Label)
	assert.Equal(t, "Average", *metricDataQuery[0].MetricStat.Stat)
	assert.Equal(t, metricName1, *metricDataQuery[0].MetricStat.Metric.MetricName)
	assert.Equal(t, namespace, *metricDataQuery[0].MetricStat.Metric.Namespace)
}

func TestCalculateRate(t *testing.T) {
	resultMetricsetFields := common.MapStr{
		"network.in.bytes":    1367316.0,
		"network.out.bytes":   427380.0,
		"network.in.packets":  2895.0,
		"network.out.packets": 2700.0,
		"diskio.read.bytes":   300.0,
		"diskio.write.bytes":  600.0,
		"diskio.read.count":   30.0,
		"diskio.write.count":  60.0,
	}

	cases := []struct {
		rateMetricName          string
		rateMetricValue         float64
		rateMetricValueDetailed float64
	}{
		{
			"network.in.bytes_per_sec",
			4557.72,
			22788.6,
		},
		{
			"network.out.bytes_per_sec",
			1424.6,
			7123.0,
		},
		{
			"network.in.packets_per_sec",
			9.65,
			48.25,
		},
		{
			"network.out.packets_per_sec",
			9.0,
			45.0,
		},
		{
			"diskio.read.bytes_per_sec",
			1.0,
			5.0,
		},
		{
			"diskio.write.bytes_per_sec",
			2.0,
			10.0,
		},
		{
			"diskio.read.count_per_sec",
			0.1,
			0.5,
		},
		{
			"diskio.write.count_per_sec",
			0.2,
			1.0,
		},
	}

	calculateRate(resultMetricsetFields, "disabled")
	for _, c := range cases {
		output, err := resultMetricsetFields.GetValue(c.rateMetricName)
		assert.NoError(t, err)
		assert.Equal(t, c.rateMetricValue, output)
	}

	calculateRate(resultMetricsetFields, "enabled")
	for _, c := range cases {
		output, err := resultMetricsetFields.GetValue(c.rateMetricName)
		assert.NoError(t, err)
		assert.Equal(t, c.rateMetricValueDetailed, output)
	}
}
