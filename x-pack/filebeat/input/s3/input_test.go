// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3iface"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
)

// MockS3Client struct is used for unit tests.
type MockS3Client struct {
	s3iface.S3API
}

var s3LogString = "36c1f05b76016b78528454e6e0c60e2b7ff7aa20c0a5e4c748276e5b0a2debd2 test-s3-ks [20/Jun/2019:04:07:48 +0000] 97.118.27.161 arn:aws:iam::627959692251:user/kaiyan.sheng@elastic.co 5141F2225A070122 REST.HEAD.OBJECT Screen%2BShot%2B2019-02-21%2Bat%2B2.15.50%2BPM.png"

func (m *MockS3Client) GetObjectRequest(input *s3.GetObjectInput) s3.GetObjectRequest {
	logBody := ioutil.NopCloser(bytes.NewReader([]byte(s3LogString)))
	return s3.GetObjectRequest{
		Request: &awssdk.Request{
			Data: &s3.GetObjectOutput{
				Body: logBody,
			},
		},
	}
}

func TestGetRegionFromQueueURL(t *testing.T) {
	queueURL := "https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs"
	regionName, err := getRegionFromQueueURL(queueURL)
	assert.NoError(t, err)
	assert.Equal(t, "us-east-1", regionName)
}

func TestHandleMessage(t *testing.T) {
	cases := []struct {
		title           string
		message         sqs.Message
		expectedS3Infos []s3Info
	}{
		{
			"sqs message with event source aws:s3 and event name ObjectCreated:Put",
			sqs.Message{
				Body: awssdk.String("{\"Records\":[{\"eventSource\":\"aws:s3\",\"awsRegion\":\"ap-southeast-1\",\"eventTime\":\"2019-06-21T16:16:54.629Z\",\"eventName\":\"ObjectCreated:Put\",\"s3\":{\"configurationId\":\"object-created-event\",\"bucket\":{\"name\":\"test-s3-ks-2\",\"arn\":\"arn:aws:s3:::test-s3-ks-2\"},\"object\":{\"key\":\"server-access-logging2019-06-21-16-16-54-E68E4316CEB285AA\"}}}]}"),
			},
			[]s3Info{
				{
					name: "test-s3-ks-2",
					key:  "server-access-logging2019-06-21-16-16-54-E68E4316CEB285AA",
				},
			},
		},
		{
			"sqs message with event source aws:s3 and event name ObjectCreated:Delete",
			sqs.Message{
				Body: awssdk.String("{\"Records\":[{\"eventSource\":\"aws:s3\",\"awsRegion\":\"ap-southeast-1\",\"eventTime\":\"2019-06-21T16:16:54.629Z\",\"eventName\":\"ObjectCreated:Delete\",\"s3\":{\"configurationId\":\"object-created-event\",\"bucket\":{\"name\":\"test-s3-ks-2\",\"arn\":\"arn:aws:s3:::test-s3-ks-2\"},\"object\":{\"key\":\"server-access-logging2019-06-21-16-16-54-E68E4316CEB285AA\"}}}]}"),
			},
			[]s3Info{},
		},
		{
			"sqs message with event source aws:ec2 and event name ObjectCreated:Put",
			sqs.Message{
				Body: awssdk.String("{\"Records\":[{\"eventSource\":\"aws:ec2\",\"awsRegion\":\"ap-southeast-1\",\"eventTime\":\"2019-06-21T16:16:54.629Z\",\"eventName\":\"ObjectCreated:Put\",\"s3\":{\"configurationId\":\"object-created-event\",\"bucket\":{\"name\":\"test-s3-ks-2\",\"arn\":\"arn:aws:s3:::test-s3-ks-2\"},\"object\":{\"key\":\"server-access-logging2019-06-21-16-16-54-E68E4316CEB285AA\"}}}]}"),
			},
			[]s3Info{},
		},
	}
	for _, c := range cases {
		s3Info, err := handleMessage(c.message)
		assert.NoError(t, err)
		assert.Equal(t, len(c.expectedS3Infos), len(s3Info))
		if len(s3Info) > 0 {
			assert.Equal(t, c.expectedS3Infos[0].key, s3Info[0].key)
			assert.Equal(t, c.expectedS3Infos[0].name, s3Info[0].name)
		}
	}
}

func TestReadS3Object(t *testing.T) {
	mockSvc := &MockS3Client{}
	s3Info := []s3Info{
		{
			name: "test-s3-ks-2",
			key:  "log2019-06-21-16-16-54",
		},
	}
	events, err := readS3Object(mockSvc, s3Info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
	bucketName, err := events[0].Fields.GetValue("log.source.bucketName")
	objectKey, err := events[0].Fields.GetValue("log.source.objectKey")
	message, err := events[0].Fields.GetValue("message")
	assert.NoError(t, err)
	assert.Equal(t, "test-s3-ks-2", bucketName.(string))
	assert.Equal(t, "log2019-06-21-16-16-54", objectKey.(string))
	assert.Equal(t, s3LogString, message.(string))
}
