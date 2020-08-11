// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
)

func TestGetAWSCredentials(t *testing.T) {
	inputConfig := ConfigAWS{
		AccessKeyID:     "123",
		SecretAccessKey: "abc",
		SessionToken:    "fake-session-token",
	}
	awsConfig, err := GetAWSCredentials(inputConfig)
	assert.NoError(t, err)

	retrievedAWSConfig, err := awsConfig.Credentials.Retrieve()
	assert.NoError(t, err)

	assert.Equal(t, inputConfig.AccessKeyID, retrievedAWSConfig.AccessKeyID)
	assert.Equal(t, inputConfig.SecretAccessKey, retrievedAWSConfig.SecretAccessKey)
	assert.Equal(t, inputConfig.SessionToken, retrievedAWSConfig.SessionToken)
}

func TestEnrichAWSConfigWithEndpoint(t *testing.T) {
	cases := []struct {
		title             string
		endpoint          string
		serviceName       string
		region            string
		awsConfig         awssdk.Config
		expectedAWSConfig awssdk.Config
	}{
		{
			"endpoint and serviceName given",
			"amazonaws.com",
			"ec2",
			"",
			awssdk.Config{},
			awssdk.Config{
				EndpointResolver: awssdk.ResolveWithEndpointURL("https://ec2.amazonaws.com"),
			},
		},
		{
			"endpoint, serviceName and region given",
			"amazonaws.com",
			"cloudwatch",
			"us-west-1",
			awssdk.Config{},
			awssdk.Config{
				EndpointResolver: awssdk.ResolveWithEndpointURL("https://cloudwatch.us-west-1.amazonaws.com"),
			},
		},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			enrichedAWSConfig := EnrichAWSConfigWithEndpoint(c.endpoint, c.serviceName, c.region, c.awsConfig)
			assert.Equal(t, c.expectedAWSConfig, enrichedAWSConfig)
		})
	}
}
