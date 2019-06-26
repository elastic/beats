// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/elastic/beats/libbeat/logp"

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

var s3LogString = "36c1f test-s3-ks [20/Jun/2019:04:07:48 +0000] 97.118.27.161 arn:aws:iam::627959692251:user/kaiyan.sheng@elastic.co 5141F2225A070122 REST.HEAD.OBJECT Screen%2BShot%2B2019-02-21%2Bat%2B2.15.50%2BPM.png"

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
		bucketNames     []string
		expectedS3Infos []s3Info
	}{
		{
			"sqs message with event source aws:s3 and event name ObjectCreated:Put",
			sqs.Message{
				Body: awssdk.String("{\"Records\":[{\"eventSource\":\"aws:s3\",\"awsRegion\":\"ap-southeast-1\",\"eventTime\":\"2019-06-21T16:16:54.629Z\",\"eventName\":\"ObjectCreated:Put\",\"s3\":{\"configurationId\":\"object-created-event\",\"bucket\":{\"name\":\"test-s3-ks-2\",\"arn\":\"arn:aws:s3:::test-s3-ks-2\"},\"object\":{\"key\":\"server-access-logging2019-06-21-16-16-54-E68E4316CEB285AA\"}}}]}"),
			},
			[]string{},
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
			[]string{},
			[]s3Info{},
		},
		{
			"sqs message with event source aws:ec2 and event name ObjectCreated:Put",
			sqs.Message{
				Body: awssdk.String("{\"Records\":[{\"eventSource\":\"aws:ec2\",\"awsRegion\":\"ap-southeast-1\",\"eventTime\":\"2019-06-21T16:16:54.629Z\",\"eventName\":\"ObjectCreated:Put\",\"s3\":{\"configurationId\":\"object-created-event\",\"bucket\":{\"name\":\"test-s3-ks-2\",\"arn\":\"arn:aws:s3:::test-s3-ks-2\"},\"object\":{\"key\":\"server-access-logging2019-06-21-16-16-54-E68E4316CEB285AA\"}}}]}"),
			},
			[]string{},
			[]s3Info{},
		},
		{
			"sqs message with right bucketNames",
			sqs.Message{
				Body: awssdk.String("{\"Records\":[{\"eventSource\":\"aws:s3\",\"awsRegion\":\"ap-southeast-1\",\"eventTime\":\"2019-06-21T16:16:54.629Z\",\"eventName\":\"ObjectCreated:Put\",\"s3\":{\"configurationId\":\"object-created-event\",\"bucket\":{\"name\":\"test-s3-ks-2\",\"arn\":\"arn:aws:s3:::test-s3-ks-2\"},\"object\":{\"key\":\"server-access-logging2019-06-21-16-16-54-E68E4316CEB285AA\"}}}]}"),
			},
			[]string{"ap-southeast-1"},
			[]s3Info{
				{
					name: "test-s3-ks-2",
					key:  "server-access-logging2019-06-21-16-16-54-E68E4316CEB285AA",
				},
			},
		},
		{
			"sqs message with wrong bucketNames",
			sqs.Message{
				Body: awssdk.String("{\"Records\":[{\"eventSource\":\"aws:s3\",\"awsRegion\":\"ap-southeast-1\",\"eventTime\":\"2019-06-21T16:16:54.629Z\",\"eventName\":\"ObjectCreated:Put\",\"s3\":{\"configurationId\":\"object-created-event\",\"bucket\":{\"name\":\"test-s3-ks-2\",\"arn\":\"arn:aws:s3:::test-s3-ks-2\"},\"object\":{\"key\":\"server-access-logging2019-06-21-16-16-54-E68E4316CEB285AA\"}}}]}"),
			},
			[]string{"us-west-1"},
			[]s3Info{},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			s3Info, err := handleMessage(c.message, c.bucketNames)
			assert.NoError(t, err)
			assert.Equal(t, len(c.expectedS3Infos), len(s3Info))
			if len(s3Info) > 0 {
				assert.Equal(t, c.expectedS3Infos[0].key, s3Info[0].key)
				assert.Equal(t, c.expectedS3Infos[0].name, s3Info[0].name)
			}
		})
	}
}

func TestReadS3Object(t *testing.T) {
	p := &Input{
		started: false,
		logger:  logp.NewLogger(inputName),
	}

	mockSvc := &MockS3Client{}
	s3Info := []s3Info{
		{
			name:   "test-s3-ks-2",
			key:    "log2019-06-21-16-16-54",
			region: "us-west-1",
		},
	}
	events, err := p.readS3Object(mockSvc, s3Info)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))

	bucketName, err := events[0].Fields.GetValue("aws.s3.bucket_name")
	assert.NoError(t, err)
	assert.Equal(t, "test-s3-ks-2", bucketName.(string))

	objectKey, err := events[0].Fields.GetValue("aws.s3.object_key")
	assert.NoError(t, err)
	assert.Equal(t, "log2019-06-21-16-16-54", objectKey.(string))

	cloudProvider, err := events[0].Fields.GetValue("cloud.provider")
	assert.NoError(t, err)
	assert.Equal(t, "aws", cloudProvider)

	region, err := events[0].Fields.GetValue("cloud.region")
	assert.NoError(t, err)
	assert.Equal(t, "us-west-1", region)

	message, err := events[0].Fields.GetValue("message")
	assert.NoError(t, err)
	assert.Equal(t, s3LogString, message.(string))
}

func TestConstructObjectURL(t *testing.T) {
	cases := []struct {
		title             string
		s3Info            s3Info
		expectedObjectURL string
	}{
		{"construct with object in s3",
			s3Info{
				name:   "test-1",
				key:    "log2019-06-21-16-16-54",
				region: "us-west-1",
			},
			"https://test-1.s3-us-west-1.amazonaws.com/log2019-06-21-16-16-54",
		},
		{"construct with object in a folder of s3",
			s3Info{
				name:   "test-2",
				key:    "test-folder-1/test-log-1.txt",
				region: "us-east-1",
			},
			"https://test-2.s3-us-east-1.amazonaws.com/test-folder-1/test-log-1.txt",
		},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			objectURL := constructObjectURL(c.s3Info)
			assert.Equal(t, c.expectedObjectURL, objectURL)
		})
	}
}
