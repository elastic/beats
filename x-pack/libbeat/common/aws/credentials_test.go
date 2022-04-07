// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"net/http"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
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
	awsConfig, err := GetAWSCredentials(inputConfig)
	assert.NoError(t, err)

	retrievedAWSConfig, err := awsConfig.Credentials.Retrieve(context.Background())
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
				EndpointResolverWithOptions: getEndpointResolverWithOptionsFunc("https://ec2.amazonaws.com"),
			},
		},
		{
			"endpoint, serviceName and region given",
			"amazonaws.com",
			"cloudwatch",
			"us-west-1",
			awssdk.Config{},
			awssdk.Config{
				EndpointResolverWithOptions: getEndpointResolverWithOptionsFunc("https://cloudwatch.us-west-1.amazonaws.com"),
			},
		},
		{
			"full URI endpoint",
			"https://s3.test.com:9000",
			"s3",
			"",
			awssdk.Config{},
			awssdk.Config{
				EndpointResolverWithOptions: getEndpointResolverWithOptionsFunc("https://s3.test.com:9000"),
			},
		},
		{
			"full non HTTPS URI endpoint",
			"http://testobjects.com:9000",
			"s3",
			"",
			awssdk.Config{},
			awssdk.Config{
				EndpointResolverWithOptions: getEndpointResolverWithOptionsFunc("http://testobjects.com:9000"),
			},
		},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			enrichedAWSConfig, err := EnrichAWSConfigWithEndpoint(c.endpoint, c.serviceName, c.region, c.awsConfig)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, c.expectedAWSConfig, enrichedAWSConfig)
		})
	}
}

func getEndpointResolverWithOptionsFunc(e string) awssdk.EndpointResolverWithOptionsFunc {
	return func(service, region string, options ...interface{}) (awssdk.Endpoint, error) {
		return awssdk.Endpoint{URL: e}, nil
	}
}

func TestCreateServiceName(t *testing.T) {
	cases := []struct {
		title               string
		serviceName         string
		fips_enabled        bool
		region              string
		expectedServiceName string
	}{
		{
			"S3 - non-fips - us-east-1",
			"s3",
			false,
			"us-east-1",
			"s3",
		},
		{
			"S3 - non-fips - us-gov-east-1",
			"s3",
			false,
			"us-gov-east-1",
			"s3",
		},
		{
			"S3 - fips - us-gov-east-1",
			"s3",
			true,
			"us-gov-east-1",
			"s3-fips",
		},
		{
			"EC2 - fips - us-gov-east-1",
			"ec2",
			true,
			"us-gov-east-1",
			"ec2",
		},
		{
			"EC2 - fips - us-east-1",
			"ec2",
			true,
			"us-east-1",
			"ec2-fips",
		},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			serviceName := CreateServiceName(c.serviceName, c.fips_enabled, c.region)
			assert.Equal(t, c.expectedServiceName, serviceName)
		})
	}
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
