// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

func TestInitializeAWSConfig(t *testing.T) {
	inputConfig := ConfigAWS{
		AccessKeyID:     "123",
		SecretAccessKey: "abc",
		TLS: &tlscommon.Config{
			VerificationMode: 1,
		},
		ProxyUrl: "http://proxy:3128",
	}
	awsConfig, err := InitializeAWSConfig(inputConfig)
	assert.NoError(t, err)

	retrievedAWSConfig, err := awsConfig.Credentials.Retrieve(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, inputConfig.AccessKeyID, retrievedAWSConfig.AccessKeyID)
	assert.Equal(t, inputConfig.SecretAccessKey, retrievedAWSConfig.SecretAccessKey)
	assert.Equal(t, true, awsConfig.HTTPClient.(*http.Client).Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify)
	assert.NotNil(t, awsConfig.HTTPClient.(*http.Client).Transport.(*http.Transport).Proxy)
}

func TestGetAWSCredentials(t *testing.T) {
	inputConfig := ConfigAWS{
		AccessKeyID:     "123",
		SecretAccessKey: "abc",
		SessionToken:    "fake-session-token",
	}
	awsConfig, err := getAWSCredentials(inputConfig)
	assert.NoError(t, err)

	retrievedAWSConfig, err := awsConfig.Credentials.Retrieve(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, inputConfig.AccessKeyID, retrievedAWSConfig.AccessKeyID)
	assert.Equal(t, inputConfig.SecretAccessKey, retrievedAWSConfig.SecretAccessKey)
	assert.Equal(t, inputConfig.SessionToken, retrievedAWSConfig.SessionToken)
}

func TestDefaultRegion(t *testing.T) {
	cases := []struct {
		title          string
		region         string
		expectedRegion string
	}{
		{
			"No default region set",
			"",
			"us-east-1",
		},
		{
			"us-west-1 region set as default",
			"us-west-1",
			"us-west-1",
		},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			inputConfig := ConfigAWS{
				AccessKeyID:     "123",
				SecretAccessKey: "abc",
			}
			if c.region != "" {
				inputConfig.DefaultRegion = c.region
			}
			awsConfig, err := InitializeAWSConfig(inputConfig)
			assert.NoError(t, err)
			assert.Equal(t, c.expectedRegion, awsConfig.Region)
		})
	}
}
