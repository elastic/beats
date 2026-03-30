// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

func TestInitializeAWSConfigCloudConnectors(t *testing.T) {
	t.Setenv(CloudConnectorsGlobalRoleEnvVar, "arn:aws:iam::999999999999:role/elastic-global-role")
	t.Setenv(CloudConnectorsJWTPathEnvVar, "/path/token")
	t.Setenv(CloudConnectorsCloudResourceIDEnvVar, "abc123")

	inputConfig := ConfigAWS{
		RoleArn:            "arn:aws:iam::123456789012:role/customer-role",
		ExternalID:         "external-id-456",
		UseCloudConnectors: true,
	}

	awsConfig, err := InitializeAWSConfig(inputConfig, logptest.NewTestingLogger(t, ""))
	assert.NoError(t, err)

	// we cannot append to APIOptions at this point (and mock the chain responses)
	// because a copy of config has already been passed to each sts client.
	// So lets just check that .Credentials is CredentialsCache (so cloud connectors init was run).
	c, isCredCache := awsConfig.Credentials.(*aws.CredentialsCache)
	require.True(t, isCredCache)
	require.NotNil(t, c)
}

func TestInitializeAWSConfig(t *testing.T) {
	inputConfig := ConfigAWS{
		AccessKeyID:     "123",
		SecretAccessKey: "abc",
		TLS: &tlscommon.Config{
			VerificationMode: 1,
		},
		ProxyUrl: "http://proxy:3128",
	}
	awsConfig, err := InitializeAWSConfig(inputConfig, logptest.NewTestingLogger(t, ""))
	assert.NoError(t, err)

	retrievedAWSConfig, err := awsConfig.Credentials.Retrieve(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, inputConfig.AccessKeyID, retrievedAWSConfig.AccessKeyID)
	assert.Equal(t, inputConfig.SecretAccessKey, retrievedAWSConfig.SecretAccessKey)
	assert.True(t, awsConfig.HTTPClient.(*http.Client).Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify) //nolint:errcheck // no need in test
	assert.NotNil(t, awsConfig.HTTPClient.(*http.Client).Transport.(*http.Transport).Proxy)                            //nolint:errcheck // no need in test
}

func TestGetAWSCredentials(t *testing.T) {
	inputConfig := ConfigAWS{
		AccessKeyID:     "123",
		SecretAccessKey: "abc",
		SessionToken:    "fake-session-token",
	}
	awsConfig, err := getAWSCredentials(inputConfig, logptest.NewTestingLogger(t, ""))
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
			awsConfig, err := InitializeAWSConfig(inputConfig, logptest.NewTestingLogger(t, ""))
			assert.NoError(t, err)
			assert.Equal(t, c.expectedRegion, awsConfig.Region)
		})
	}
}
