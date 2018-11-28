// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/metricbeat/mb"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

type MockEC2Client struct {
	ec2iface.EC2API
}

type MockCloudWatchClient struct {
	cloudwatchiface.CloudWatchAPI
}

var regionName = "us-west-1"

func (m *MockEC2Client) DescribeRegions(input *ec2.DescribeRegionsInput) (output *ec2.DescribeRegionsOutput, err error) {
	output = &ec2.DescribeRegionsOutput{
		Regions: []*ec2.Region{
			&ec2.Region{
				RegionName: &regionName,
			},
		},
	}
	return
}

func (m *MockEC2Client) DescribeInstances(input *ec2.DescribeInstancesInput) (output *ec2.DescribeInstancesOutput, err error) {
	instance1 := &ec2.Instance{InstanceId: aws.String("i-123"), InstanceType: aws.String("t1.medium")}
	instance2 := &ec2.Instance{InstanceId: aws.String("i-456"), InstanceType: aws.String("t2.micro")}
	output = &ec2.DescribeInstancesOutput{
		Reservations: []*ec2.Reservation{
			&ec2.Reservation{
				Instances: []*ec2.Instance{
					instance1,
					instance2,
				},
			},
		},
	}
	return
}

func (m *MockCloudWatchClient) GetMetricData(input *cloudwatch.GetMetricDataInput) (output *cloudwatch.GetMetricDataOutput, err error) {
	id := "cpu1"
	label := "CPUUtilization"
	value := 0.25
	output = &cloudwatch.GetMetricDataOutput{
		MetricDataResults: []*cloudwatch.MetricDataResult{
			&cloudwatch.MetricDataResult{
				Id:     &id,
				Label:  &label,
				Values: []*float64{&value},
			},
		},
	}
	return
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
	instanceIDs, output, err := getInstancesPerRegion(mockSvc)
	if err != nil {
		fmt.Println("failed getInstancesPerRegion: ", err)
		t.FailNow()
	}

	assert.Equal(t, 2, len(instanceIDs))
	assert.Equal(t, 1, len(output))
	reservations := output[0].Reservations
	assert.Equal(t, 1, len(reservations))
	instances := reservations[0].Instances
	assert.Equal(t, 2, len(instances))
	assert.Equal(t, "i-123", instanceIDs[0])
	assert.Equal(t, "i-456", instanceIDs[1])
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
	assert.Equal(t, 0.25, *getMetricDataOutput.MetricDataResults[0].Values[0])
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
		assert.Equal(t, 4, len(events))
	}

	for _, event := range events {
		checkSpecificMetric("cpu_utilization", "float64", event, t)
		checkSpecificMetric("instance.id", "string", event, t)
	}
}

func checkSpecificMetric(metricName string, expectFormat string, event mb.Event, t *testing.T) {
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
			} else {
				if userString != "i-456" {
					assert.Equal(t, userString, "i-123")
				} else {
					assert.Equal(t, userString, "i-456")
				}
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

	sess, err := session.NewSession()
	if err != nil {
		fmt.Println("NewSession failed: ", err)
		return nil, err
	}

	stsSvc := sts.New(sess)
	getSessionTokenInput := sts.GetSessionTokenInput{
		SerialNumber: aws.String(os.Getenv("SERIAL_NUMBER")),
		TokenCode:    aws.String(os.Getenv("MFA_TOKEN")),
	}

	tempToken, err := stsSvc.GetSessionToken(&getSessionTokenInput)
	if err != nil {
		fmt.Println("GetSessionToken failed: ", err)
		return nil, err
	}

	accessKeyID := *tempToken.Credentials.AccessKeyId
	secretAccessKey := *tempToken.Credentials.SecretAccessKey
	sessionToken := *tempToken.Credentials.SessionToken
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
