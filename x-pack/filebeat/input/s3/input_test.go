// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3iface"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
)

// MockS3Client struct is used for unit tests.
type MockS3Client struct {
	s3iface.ClientAPI
}

// MockS3ClientErr struct is used for unit tests.
type MockS3ClientErr struct {
	s3iface.ClientAPI
}

var (
	s3LogString1 = "36c1f test-s3-ks [20/Jun/2019] 1.2.3.4 arn:aws:iam::1234:user/test@elastic.co 5141F REST.HEAD.OBJECT Screen1.png \n"
	s3LogString2 = "28kdg test-s3-ks [20/Jun/2019] 1.2.3.4 arn:aws:iam::1234:user/test@elastic.co 5A070 REST.HEAD.OBJECT Screen2.png \n"
	mockSvc      = &MockS3Client{}
	mockSvcErr   = &MockS3ClientErr{}
	info         = s3Info{
		name:   "test-s3-ks",
		key:    "log2019-06-21-16-16-54",
		region: "us-west-1",
	}
)

func (m *MockS3Client) GetObjectRequest(input *s3.GetObjectInput) s3.GetObjectRequest {
	logBody := ioutil.NopCloser(bytes.NewReader([]byte(s3LogString1 + s3LogString2)))
	httpReq, _ := http.NewRequest("", "", nil)
	return s3.GetObjectRequest{
		Request: &awssdk.Request{
			Data: &s3.GetObjectOutput{
				Body: logBody,
			},
			HTTPRequest: httpReq,
		},
	}
}

func (m *MockS3ClientErr) GetObjectRequest(input *s3.GetObjectInput) s3.GetObjectRequest {
	httpReq, _ := http.NewRequest("", "", nil)
	return s3.GetObjectRequest{
		Request: &awssdk.Request{
			Data:        &s3.GetObjectOutput{},
			HTTPRequest: httpReq,
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
		t.Run(c.title, func(t *testing.T) {
			s3Info, err := handleSQSMessage(c.message)
			assert.NoError(t, err)
			assert.Equal(t, len(c.expectedS3Infos), len(s3Info))
			if len(s3Info) > 0 {
				assert.Equal(t, c.expectedS3Infos[0].key, s3Info[0].key)
				assert.Equal(t, c.expectedS3Infos[0].name, s3Info[0].name)
			}
		})
	}
}

func TestNewS3BucketReader(t *testing.T) {
	p := &s3Input{context: &channelContext{}}
	reader, err := p.newS3BucketReader(mockSvc, info)
	assert.NoError(t, err)
	for i := 0; i < 3; i++ {
		switch i {
		case 0:
			log, err := reader.ReadString('\n')
			assert.NoError(t, err)
			assert.Equal(t, s3LogString1, log)
		case 1:
			log, err := reader.ReadString('\n')
			assert.NoError(t, err)
			assert.Equal(t, s3LogString2, log)
		case 2:
			log, err := reader.ReadString('\n')
			assert.Error(t, io.EOF, err)
			assert.Equal(t, "", log)
		}
	}
}

func TestNewS3BucketReaderErr(t *testing.T) {
	p := &s3Input{context: &channelContext{}}
	reader, err := p.newS3BucketReader(mockSvcErr, info)
	assert.Error(t, err, "s3 get object response body is empty")
	assert.Nil(t, reader)
}

func TestCreateEvent(t *testing.T) {
	p := &s3Input{context: &channelContext{}}
	errC := make(chan error)
	s3Context := &s3Context{
		refs: 1,
		errC: errC,
	}

	mockSvc := &MockS3Client{}
	s3Info := s3Info{
		name:   "test-s3-ks",
		key:    "log2019-06-21-16-16-54",
		region: "us-west-1",
		arn:    "arn:aws:s3:::test-s3-ks",
	}
	s3ObjectHash := s3ObjectHash(s3Info)

	reader, err := p.newS3BucketReader(mockSvc, s3Info)
	assert.NoError(t, err)
	var events []beat.Event
	for {
		log, err := reader.ReadString('\n')
		if log == "" {
			break
		}
		if err == io.EOF {
			event := createEvent(log, len([]byte(log)), s3Info, s3ObjectHash, s3Context)
			events = append(events, event)
			break
		}

		event := createEvent(log, len([]byte(log)), s3Info, s3ObjectHash, s3Context)
		events = append(events, event)
	}

	assert.Equal(t, 2, len(events))

	bucketName, err := events[0].Fields.GetValue("aws.s3.bucket.name")
	assert.NoError(t, err)
	assert.Equal(t, "test-s3-ks", bucketName.(string))

	objectKey, err := events[0].Fields.GetValue("aws.s3.object.key")
	assert.NoError(t, err)
	assert.Equal(t, "log2019-06-21-16-16-54", objectKey.(string))

	cloudProvider, err := events[0].Fields.GetValue("cloud.provider")
	assert.NoError(t, err)
	assert.Equal(t, "aws", cloudProvider)

	region, err := events[0].Fields.GetValue("cloud.region")
	assert.NoError(t, err)
	assert.Equal(t, "us-west-1", region)

	message1, err := events[0].Fields.GetValue("message")
	assert.NoError(t, err)
	assert.Equal(t, s3LogString1, message1.(string))

	message2, err := events[1].Fields.GetValue("message")
	assert.NoError(t, err)
	assert.Equal(t, s3LogString2, message2.(string))

	s3Context.done()
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

func TestConvertOffsetToString(t *testing.T) {
	cases := []struct {
		offset         int
		expectedString string
	}{
		{
			123,
			"000000000123",
		},
		{
			123456,
			"000000123456",
		},
		{
			123456789123,
			"123456789123",
		},
	}
	for _, c := range cases {
		output := fmt.Sprintf("%012d", c.offset)
		assert.Equal(t, c.expectedString, output)
	}

}
