// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws/external"

	"github.com/elastic/beats/libbeat/common"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/metricbeat/mb"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

var regionName = "us-west-1"

func (m *MockEC2Client) DescribeRegionsRequest(input *ec2.DescribeRegionsInput) ec2.DescribeRegionsRequest {
	return ec2.DescribeRegionsRequest{
		Request: &awssdk.Request{
			Data: &ec2.DescribeRegionsOutput{
				Regions: []ec2.Region{
					ec2.Region{
						RegionName: &regionName,
					},
				},
			},
		},
	}
}

func (m *MockEC2Client) DescribeInstancesRequest(input *ec2.DescribeInstancesInput) ec2.DescribeInstancesRequest {
	instance1 := ec2.Instance{
		InstanceId:   awssdk.String("i-123"),
		InstanceType: ec2.InstanceTypeT2Medium,
		Placement:    &ec2.Placement{AvailabilityZone: awssdk.String("us-west-1a")},
		ImageId:      awssdk.String("image-123"),
	}
	instance2 := ec2.Instance{
		InstanceId:   awssdk.String("i-456"),
		InstanceType: ec2.InstanceTypeT2Micro,
		Placement:    &ec2.Placement{AvailabilityZone: awssdk.String("us-west-1b")},
		ImageId:      awssdk.String("image-456"),
	}
	return ec2.DescribeInstancesRequest{
		Request: &awssdk.Request{
			Data: &ec2.DescribeInstancesOutput{
				Reservations: []ec2.RunInstancesOutput{
					ec2.RunInstancesOutput{Instances: []ec2.Instance{instance1, instance2}},
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
	getMetricDataOutput, err := getMetricDataPerRegion("i-123", nil, mockSvc)
	if err != nil {
		fmt.Println("failed getMetricDataPerRegion: ", err)
		t.FailNow()
	}
	assert.Equal(t, 1, len(getMetricDataOutput.MetricDataResults))
	assert.Equal(t, "cpu1", *getMetricDataOutput.MetricDataResults[0].Id)
	assert.Equal(t, "CPUUtilization", *getMetricDataOutput.MetricDataResults[0].Label)
	assert.Equal(t, 0.25, getMetricDataOutput.MetricDataResults[0].Values[0])
}

func TestFetch(t *testing.T) {
	mock := true
	tempCredentials, err := getCredentials(mock)
	if err != nil {
		fmt.Println("failed getCredentials: ", err)
		t.FailNow()
	}

	awsMetricSet := mbtest.NewReportingMetricSetV2(t, tempCredentials)
	events, errs := mbtest.ReportingFetchV2(awsMetricSet)
	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}
	t.Logf("Module: %s Metricset: %s", awsMetricSet.Module().Name(), awsMetricSet.Name())

	if mock {
		assert.Equal(t, 2, len(events))
	}

	for _, event := range events {
		checkRootField("service", event, t)
		checkMetricSetField("cpu_utilization", "float64", event, t)
		checkMetricSetField("instance.id", "string", event, t)
	}
}

func checkRootField(fieldName string, event mb.Event, t *testing.T) {
	expectedKeyValue := common.MapStr{"name": "aws"}
	if ok, err := event.RootFields.HasKey(fieldName); ok {
		assert.NoError(t, err)
		fieldValue, err := event.RootFields.GetValue(fieldName)
		assert.NoError(t, err)
		assert.Equal(t, expectedKeyValue, fieldValue)
	}
}

func checkMetricSetField(metricName string, expectFormat string, event mb.Event, t *testing.T) {
	if ok, err := event.MetricSetFields.HasKey(metricName); ok {
		assert.NoError(t, err)
		metricValue, err := event.MetricSetFields.GetValue(metricName)
		assert.NoError(t, err)
		switch expectFormat {
		case "float64":
			if userPercentFloat, ok := metricValue.(float64); !ok {
				fmt.Println("failed: userPercentFloat = ", userPercentFloat)
				t.Fail()
			} else {
				assert.True(t, userPercentFloat >= 0)
				fmt.Println("succeed: userPercentFloat = ", userPercentFloat)
			}
		case "string":
			if userString, ok := metricValue.(string); !ok {
				fmt.Println("failed: userString = ", userString)
				t.Fail()
			}
		}
	}
}

func getCredentials(mock bool) (map[string]interface{}, error) {
	if mock {
		creds := map[string]interface{}{
			"module":     "aws",
			"metricsets": []string{"ec2"},
			"mock":       "true",
		}
		return creds, nil
	}

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		fmt.Println("failed to load config: ", err.Error())
	}

	stsSvc := sts.New(cfg)
	getSessionTokenInput := sts.GetSessionTokenInput{
		SerialNumber: awssdk.String(os.Getenv("SERIAL_NUMBER")),
		TokenCode:    awssdk.String(os.Getenv("MFA_TOKEN")),
	}

	req := stsSvc.GetSessionTokenRequest(&getSessionTokenInput)
	tempToken, err := req.Send()
	if err != nil {
		fmt.Println("GetSessionToken failed: ", err)
		return nil, err
	}

	accessKeyID := *tempToken.Credentials.AccessKeyId
	secretAccessKey := *tempToken.Credentials.SecretAccessKey
	sessionToken := *tempToken.Credentials.SessionToken
	os.Setenv("AWS_ACCESS_KEY_ID", accessKeyID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", secretAccessKey)
	os.Setenv("AWS_SESSION_TOKEN", sessionToken)

	creds := map[string]interface{}{
		"module":            "aws",
		"metricsets":        []string{"ec2"},
		"mock":              "false",
		"access_key_id":     accessKeyID,
		"secret_access_key": secretAccessKey,
		"session_token":     sessionToken,
	}
	return creds, nil
}
