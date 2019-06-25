// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package sqs

import (
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/sqsiface"
	"github.com/stretchr/testify/assert"
)

// MockSQSClient struct is used for unit tests.
type MockSQSClient struct {
	sqsiface.ClientAPI
}

func (m *MockSQSClient) ListQueuesRequest(input *sqs.ListQueuesInput) sqs.ListQueuesRequest {
	return sqs.ListQueuesRequest{
		Request: &awssdk.Request{
			Data: &sqs.ListQueuesOutput{
				QueueUrls: []string{"https://sqs.us-east-1.amazonaws.com/123/sqs1", "https://sqs.us-east-1.amazonaws.com/123/sqs2"},
			},
		},
	}
}

func TestGetQueueUrls(t *testing.T) {
	mockSvc := &MockSQSClient{}
	queueUrls, err := getQueueUrls(mockSvc)
	assert.NoError(t, err)
	assert.Equal(t, []string{"https://sqs.us-east-1.amazonaws.com/123/sqs1", "https://sqs.us-east-1.amazonaws.com/123/sqs2"}, queueUrls)
}
