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

type mockEC2Client struct {
	ec2iface.EC2API
}

type mockCloudWatchClient struct {
	cloudwatchiface.CloudWatchAPI
}

func (m *mockEC2Client) DescribeInstances(input *ec2.DescribeInstancesInput) (output *ec2.DescribeInstancesOutput, err error) {
	instance1 := &ec2.Instance{InstanceId: aws.String("i-123")}
	instance2 := &ec2.Instance{InstanceId: aws.String("i-456")}
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

func (m *mockCloudWatchClient) GetMetricData(input *cloudwatch.GetMetricDataInput) (output *cloudwatch.GetMetricDataOutput, err error) {
	id := "m1"
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

func TestGetInstanceIDs(t *testing.T) {
	mockSvc := &mockEC2Client{}
	instanceIDs, err := getInstancesPerRegion(mockSvc)
	if err != nil {
		fmt.Println("failed getInstancesPerRegion: ", err)
		t.FailNow()
	}
	assert.Equal(t, 2, len(instanceIDs))
	assert.Equal(t, "i-123", instanceIDs[0])
	assert.Equal(t, "i-456", instanceIDs[1])
}

func TestGetMetricDataPerRegion(t *testing.T) {
	mockSvc := &mockCloudWatchClient{}
	getMetricDataOutput, err := getMetricDataPerRegion("i-123", nil, mockSvc)
	if err != nil {
		fmt.Println("failed getMetricDataPerRegion: ", err)
		t.FailNow()
	}
	assert.Equal(t, 1, len(getMetricDataOutput.MetricDataResults))
	assert.Equal(t, "m1", *getMetricDataOutput.MetricDataResults[0].Id)
	assert.Equal(t, "CPUUtilization", *getMetricDataOutput.MetricDataResults[0].Label)
	assert.Equal(t, 0.25, *getMetricDataOutput.MetricDataResults[0].Values[0])
}

func TestFetch(t *testing.T) {
	os.Setenv("MFA_TOKEN", "mfa_token")
	os.Setenv("SERIAL_NUMBER", "serial_number")
	tempCredentials, err := getTemporaryTokenUsingMFA()
	if err != nil {
		fmt.Println("failed getTemporaryTokenUsingMFA: ", err)
		t.FailNow()
	}

	awsMetricSet := mbtest.NewReportingMetricSetV2(t, tempCredentials)
	events, errs := mbtest.ReportingFetchV2(awsMetricSet)
	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}
	t.Logf("Module: %s Metricset: %s", awsMetricSet.Module().Name(), awsMetricSet.Name())
	for _, event := range events {
		checkSpecificMetric("cpu_utilization", event, t)
		checkSpecificMetric("cpu_credit_usage", event, t)
		checkSpecificMetric("cpu_credit_balance", event, t)
	}
}

func checkSpecificMetric(metricName string, event mb.Event, t *testing.T) {
	if ok, err := event.MetricSetFields.HasKey(metricName); ok {
		assert.NoError(t, err)
		cpuUtilization, err := event.MetricSetFields.GetValue(metricName)
		assert.NoError(t, err)
		if userPercentFloat, ok := cpuUtilization.(float64); !ok {
			fmt.Println("failed: userPercentFloat = ", userPercentFloat)
			t.Fail()
		} else {
			assert.True(t, userPercentFloat >= 0)
			fmt.Println("succeed: userPercentFloat = ", userPercentFloat)
		}
	}
}

func getTemporaryTokenUsingMFA() (map[string]interface{}, error) {
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

	accessKeyId := *tempToken.Credentials.AccessKeyId
	secretAccessKey := *tempToken.Credentials.SecretAccessKey
	sessionToken := *tempToken.Credentials.SessionToken
	fmt.Println("accessKeyId = ", accessKeyId)
	fmt.Println("secretAccessKey = ", secretAccessKey)
	fmt.Println("sessionToken = ", sessionToken)
	creds := map[string]interface{}{
		"module":            "aws",
		"metricsets":        []string{"ec2"},
		"access_key_id":     accessKeyId,
		"secret_access_key": secretAccessKey,
		"session_token":     sessionToken,
	}
	return creds, nil
}
