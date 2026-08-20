// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestSQSDiscoveryV2_getS3Notifications_S3Direct(t *testing.T) {
	body := `{"Records":[{"eventSource":"aws:s3","eventName":"ObjectCreated:Put","awsRegion":"us-east-1","s3":{"bucket":{"name":"my-bucket","arn":"arn:aws:s3:::my-bucket"},"object":{"key":"logs/2024/file.log"}}}]}`
	d := &sqsDiscoveryV2{log: logptest.NewTestingLogger(t, t.Name())}

	events, err := d.getS3Notifications(body)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "my-bucket", events[0].S3.Bucket.Name)
	assert.Equal(t, "logs/2024/file.log", events[0].S3.Object.Key)
	assert.Equal(t, "us-east-1", events[0].AWSRegion)
}

func TestSQSDiscoveryV2_getS3Notifications_SNSWrapped(t *testing.T) {
	inner := `{"Records":[{"eventSource":"aws:s3","eventName":"ObjectCreated:Put","awsRegion":"eu-west-1","s3":{"bucket":{"name":"wrapped-bucket"},"object":{"key":"data.json"}}}]}`
	outer := map[string]string{
		"TopicArn": "arn:aws:sns:eu-west-1:123456789012:my-topic",
		"Message":  inner,
	}
	body, err := json.Marshal(outer)
	require.NoError(t, err)

	d := &sqsDiscoveryV2{log: logptest.NewTestingLogger(t, t.Name())}
	events, err := d.getS3Notifications(string(body))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "wrapped-bucket", events[0].S3.Bucket.Name)
	assert.Equal(t, "data.json", events[0].S3.Object.Key)
}

func TestSQSDiscoveryV2_getS3Notifications_EventBridge(t *testing.T) {
	body := `{"version":"0","id":"abc123","detail-type":"Object Created","source":"aws.s3","account":"123456789012","time":"2024-01-01T00:00:00Z","region":"us-west-2","resources":["arn:aws:s3:::eb-bucket"],"detail":{"version":"0","bucket":{"name":"eb-bucket"},"object":{"key":"eb-key.log","size":100,"etag":"abc","version-id":"","sequencer":""},"request-id":"","requester":"","source-ip-address":"","reason":""}}`

	d := &sqsDiscoveryV2{log: logptest.NewTestingLogger(t, t.Name())}
	events, err := d.getS3Notifications(body)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "eb-bucket", events[0].S3.Bucket.Name)
	assert.Equal(t, "eb-key.log", events[0].S3.Object.Key)
	assert.Equal(t, "us-west-2", events[0].AWSRegion)
}

func TestSQSDiscoveryV2_getS3Notifications_TestEvent(t *testing.T) {
	body := `{"Event":"s3:TestEvent","Records":null}`
	d := &sqsDiscoveryV2{log: logptest.NewTestingLogger(t, t.Name())}

	events, err := d.getS3Notifications(body)
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestSQSDiscoveryV2_getS3Notifications_SNSTestEvent(t *testing.T) {
	inner := `{"Event":"s3:TestEvent"}`
	outer := map[string]string{
		"TopicArn": "arn:aws:sns:us-east-1:123456789012:topic",
		"Message":  inner,
	}
	body, err := json.Marshal(outer)
	require.NoError(t, err)

	d := &sqsDiscoveryV2{log: logptest.NewTestingLogger(t, t.Name())}
	events, err := d.getS3Notifications(string(body))
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestSQSDiscoveryV2_getS3Notifications_MissingRecords(t *testing.T) {
	body := `{"someField":"value"}`
	d := &sqsDiscoveryV2{log: logptest.NewTestingLogger(t, t.Name())}

	_, err := d.getS3Notifications(body)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing Records field")
}

func TestSQSDiscoveryV2_getS3Notifications_URLUnescape(t *testing.T) {
	body := `{"Records":[{"eventSource":"aws:s3","eventName":"ObjectCreated:Put","awsRegion":"us-east-1","s3":{"bucket":{"name":"bucket"},"object":{"key":"path%3D%2Fencoded.log"}}}]}`
	d := &sqsDiscoveryV2{log: logptest.NewTestingLogger(t, t.Name())}

	events, err := d.getS3Notifications(body)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "path=/encoded.log", events[0].S3.Object.Key)
}

func TestSQSDiscoveryV2_filterRecords_NonObjectCreated(t *testing.T) {
	records := []s3EventV2{
		{EventSource: "aws:s3", EventName: "ObjectCreated:Put"},
		{EventSource: "aws:s3", EventName: "ObjectRemoved:Delete"},
	}
	records[0].S3.Bucket.Name = "bucket"
	records[0].S3.Object.Key = "keep.log"
	records[1].S3.Bucket.Name = "bucket"
	records[1].S3.Object.Key = "skip.log"

	d := &sqsDiscoveryV2{log: logptest.NewTestingLogger(t, t.Name())}
	out, err := d.filterRecords(records)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "keep.log", out[0].S3.Object.Key)
}
