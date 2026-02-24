// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestAddCloudConnectorsCredentials(t *testing.T) {
	config := ConfigAWS{
		RoleArn:                "arn:aws:iam::123456789012:role/customer-role",
		ExternalID:             "external-id-456",
		AssumeRoleDuration:     2 * time.Hour,
		AssumeRoleExpiryWindow: 10 * time.Minute,
	}
	cloudConnectorsConfig := CloudConnectorsConfig{
		ElasticGlobalRoleARN: "arn:aws:iam::999999999999:role/elastic-global-role",
		CloudResourceID:      "abcd1234",
	}
	tokenFileContent := "abc123"

	tmpDir := t.TempDir()
	pth := path.Join(tmpDir, "id_token")
	_ = os.WriteFile(path.Join(tmpDir, "id_token"), []byte(tokenFileContent), 0o644)
	cloudConnectorsConfig.IDTokenPath = pth

	// Create a base AWS config
	awsConfig := &aws.Config{
		Region:       "us-east-1",
		BaseEndpoint: aws.String("https://aws.mock"),
	}

	// Create a test logger
	logger := logptest.NewTestingLogger(t, "")

	// mock responses
	receivedCalls := 0
	awsConfig.APIOptions = append(awsConfig.APIOptions, func(stack *middleware.Stack) error {
		return stack.Finalize.Add(
			middleware.FinalizeMiddlewareFunc(
				"mock",
				func(ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
					req, is := in.Request.(*smithyhttp.Request)
					require.Truef(t, is, "request expected to be of type *smithyhttp.Request, got: %T", in.Request)
					receivedCalls++
					bd, err := io.ReadAll(req.GetStream())
					assert.NoError(t, req.RewindStream())
					assert.NoError(t, err)
					body := string(bd)

					switch receivedCalls {

					// Expect the first request to be AssumeRoleWithWebIdentity
					case 1:
						q, err := url.ParseQuery(body)
						assert.NoError(t, err)
						assert.Equal(t, "AssumeRoleWithWebIdentity", q.Get("Action"))
						assert.Equal(t, "1200", q.Get("DurationSeconds"))
						assert.Equal(t, cloudConnectorsConfig.ElasticGlobalRoleARN, q.Get("RoleArn"))
						assert.Equal(t, tokenFileContent, q.Get("WebIdentityToken"))
						return middleware.FinalizeOutput{
							Result: &sts.AssumeRoleWithWebIdentityOutput{
								Credentials: &types.Credentials{
									AccessKeyId:     aws.String("AKIAFAKEEXAMPLE00001"),
									SecretAccessKey: aws.String("FAKEwJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY1"),
									SessionToken:    aws.String("FwoGZXIvYXdzEFAaDFAKESESSIONTOKENEXAMPLE1"),
									Expiration:      aws.Time(time.Now().Add(defaultIntermediateDuration)),
								},
							},
						}, middleware.Metadata{}, nil

					// Expect the second request to be AssumeRole
					case 2:
						q, err := url.ParseQuery(body)
						assert.NoError(t, err)
						assert.Equal(t, "AssumeRole", q.Get("Action"))
						assert.Equal(t, "7200", q.Get("DurationSeconds"))
						assert.Equal(t, cloudConnectorsExternalID(cloudConnectorsConfig.CloudResourceID, config.ExternalID), q.Get("ExternalId"))
						assert.Equal(t, config.RoleArn, q.Get("RoleArn"))
						return middleware.FinalizeOutput{
							Result: &sts.AssumeRoleOutput{
								Credentials: &types.Credentials{
									AccessKeyId:     aws.String("AKIAFAKEEXAMPLE00002"),
									SecretAccessKey: aws.String("FAKEwJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY2"),
									SessionToken:    aws.String("FwoGZXIvYXdzEFAaDFAKESESSIONTOKENEXAMPLE2"),
									Expiration:      aws.Time(time.Now().Add(defaultIntermediateDuration)),
								},
							},
						}, middleware.Metadata{}, nil

					default:
						t.Fatal("unexpected aws sdk call")
						return middleware.FinalizeOutput{}, middleware.Metadata{}, fmt.Errorf("unexpected operation")
					}
				},
			),
			middleware.After,
		)
	})

	// Call the function under test
	addCloudConnectorsCredentials(
		config,
		cloudConnectorsConfig,
		awsConfig,
		logger,
	)

	// Verify that credentials provider was set
	require.NotNil(t, awsConfig.Credentials, "credentials provider should be set")

	crd, err := awsConfig.Credentials.Retrieve(t.Context())
	require.NoError(t, err)
	require.NotNil(t, crd)
	require.Equal(t, 2, receivedCalls)
}

func TestCloudConnectorsExternalID(t *testing.T) {
	assert.Equal(t, "resource1-ext-id", cloudConnectorsExternalID("resource1", "ext-id"))
	assert.Equal(t, "abc123-external-id-456", cloudConnectorsExternalID("abc123", "external-id-456"))
	assert.Equal(t, "single-", cloudConnectorsExternalID("single", "")) // format is always "resourceID-externalIDPart"
}

func TestParseCloudConnectorsConfigFromEnv(t *testing.T) {
	t.Run("happy_path", func(t *testing.T) {
		t.Setenv(CloudConnectorsGlobalRoleEnvVar, "arn:aws:iam::999999999999:role/elastic-global-role")
		t.Setenv(CloudConnectorsJWTPathEnvVar, "/path/token")
		t.Setenv(CloudConnectorsCloudResourceIDEnvVar, "abc123")

		got, err := parseCloudConnectorsConfigFromEnv()

		require.NoError(t, err)

		assert.Equal(
			t,
			CloudConnectorsConfig{
				ElasticGlobalRoleARN: "arn:aws:iam::999999999999:role/elastic-global-role",
				IDTokenPath:          "/path/token",
				CloudResourceID:      "abc123",
			},
			got,
		)
	})

	t.Run("missing config single", func(t *testing.T) {
		t.Setenv(CloudConnectorsGlobalRoleEnvVar, "arn:aws:iam::999999999999:role/elastic-global-role")
		t.Setenv(CloudConnectorsJWTPathEnvVar, "/path/token")

		got, err := parseCloudConnectorsConfigFromEnv()

		require.ErrorContains(t, err, "cloud resource id")
		assert.Equal(t, CloudConnectorsConfig{}, got)
	})

	t.Run("missing config all", func(t *testing.T) {
		got, err := parseCloudConnectorsConfigFromEnv()

		require.ErrorContains(t, err, "elastic global role")
		require.ErrorContains(t, err, "id token")
		require.ErrorContains(t, err, "cloud resource id")
		assert.Equal(t, CloudConnectorsConfig{}, got)
	})
}
