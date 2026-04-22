// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"fmt"
	"io"
	"net/http"
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
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"

	identityfederationaws "github.com/elastic/beats/v7/x-pack/libbeat/common/identityfederation/aws"
)

func TestInitializeAWSConfigIdentityFederation(t *testing.T) {
	t.Setenv(identityfederationaws.GlobalRoleARNEnvVar, "arn:aws:iam::999999999999:role/elastic-global-role")
	t.Setenv(identityfederationaws.IDTokenFileEnvVar, "/path/token")
	t.Setenv(identityfederationaws.CloudResourceIDEnvVar, "abc123")

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

// TestApplyIdentityFederationChain exercises the full two-step STS chain with mock responses,
// verifying that the correct parameters are sent to STS at each step.
func TestApplyIdentityFederationChain(t *testing.T) {
	config := ConfigAWS{
		RoleArn:                "arn:aws:iam::123456789012:role/customer-role",
		ExternalID:             "external-id-456",
		AssumeRoleDuration:     2 * time.Hour,
		AssumeRoleExpiryWindow: 10 * time.Minute,
	}

	globalRoleARN := "arn:aws:iam::999999999999:role/elastic-global-role"
	cloudResourceID := "abcd1234"
	tokenFileContent := "abc123"

	tmpDir := t.TempDir()
	pth := path.Join(tmpDir, "id_token")
	_ = os.WriteFile(pth, []byte(tokenFileContent), 0o644)

	t.Setenv(identityfederationaws.GlobalRoleARNEnvVar, globalRoleARN)
	t.Setenv(identityfederationaws.IDTokenFileEnvVar, pth)
	t.Setenv(identityfederationaws.CloudResourceIDEnvVar, cloudResourceID)

	// Create a base AWS config with a mock STS interceptor injected via APIOptions.
	// The interceptor must be set on the base config before calling applyIdentityFederationChain,
	// because each step creates its STS client from the config at that point in the chain.
	baseConfig := &aws.Config{
		Region:       "us-east-1",
		BaseEndpoint: aws.String("https://aws.mock"),
	}

	receivedCalls := 0
	baseConfig.APIOptions = append(baseConfig.APIOptions, func(stack *middleware.Stack) error {
		return stack.Finalize.Add(
			middleware.FinalizeMiddlewareFunc(
				"mock",
				func(ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
					req, is := in.Request.(*smithyhttp.Request)
					require.Truef(t, is, "expected *smithyhttp.Request, got: %T", in.Request)
					receivedCalls++
					bd, err := io.ReadAll(req.GetStream())
					assert.NoError(t, req.RewindStream())
					assert.NoError(t, err)
					body := string(bd)

					switch receivedCalls {

					// Step 1: AssumeRoleWithWebIdentity → Elastic Global Role
					case 1:
						q, err := url.ParseQuery(body)
						assert.NoError(t, err)
						assert.Equal(t, "AssumeRoleWithWebIdentity", q.Get("Action"))
						assert.Equal(t, "1200", q.Get("DurationSeconds")) // defaultIntermediateDuration
						assert.Equal(t, globalRoleARN, q.Get("RoleArn"))
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

					// Step 2: AssumeRole → customer remote role
					case 2:
						q, err := url.ParseQuery(body)
						assert.NoError(t, err)
						assert.Equal(t, "AssumeRole", q.Get("Action"))
						assert.Equal(t, "7200", q.Get("DurationSeconds")) // 2 * time.Hour
						assert.Equal(t, cloudResourceID+"-"+config.ExternalID, q.Get("ExternalId"))
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

	err := applyIdentityFederationChain(config, baseConfig, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	require.NotNil(t, baseConfig.Credentials, "credentials provider should be set")
	crd, err := baseConfig.Credentials.Retrieve(t.Context())
	require.NoError(t, err)
	require.NotNil(t, crd)
	require.Equal(t, 2, receivedCalls)
}

func TestApplyIdentityFederationChainValidation(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	t.Run("missing cloud resource id", func(t *testing.T) {
		t.Setenv(identityfederationaws.GlobalRoleARNEnvVar, "arn:aws:iam::999999999999:role/elastic-global-role")
		t.Setenv(identityfederationaws.IDTokenFileEnvVar, "/path/token")
		// CloudResourceIDEnvVar intentionally not set

		err := applyIdentityFederationChain(ConfigAWS{}, &aws.Config{}, logger)
		require.ErrorContains(t, err, "cloud resource id")
	})

	t.Run("missing all env vars", func(t *testing.T) {
		err := applyIdentityFederationChain(ConfigAWS{}, &aws.Config{}, logger)
		require.ErrorContains(t, err, "elastic global role")
		require.ErrorContains(t, err, "id token")
		require.ErrorContains(t, err, "cloud resource id")
	})
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
