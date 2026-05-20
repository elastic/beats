// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package identityfederation

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAWSFormatExternalID(t *testing.T) {
	assert.Equal(t, "resource1-ext-id", AWSFormatExternalID("resource1", "ext-id"))
	assert.Equal(t, "abc123-external-id-456", AWSFormatExternalID("abc123", "external-id-456"))
	assert.Equal(t, "single-", AWSFormatExternalID("single", ""))
}

// TestAWSConfigRoleChaining verifies that a multi-step chain runs each step in
// order and that the final config carries the last step's credentials.
func TestAWSConfigRoleChaining(t *testing.T) {
	globalRoleARN := "arn:aws:iam::999999999999:role/elastic-global-role"
	remoteRoleARN := "arn:aws:iam::123456789012:role/customer-role"
	resourceID := "abcd1234"
	externalIDPart := "external-id-456"
	tokenFileContent := "abc123"

	tmpDir := t.TempDir()
	tokenPath := path.Join(tmpDir, "id_token")
	require.NoError(t, os.WriteFile(tokenPath, []byte(tokenFileContent), 0o644))

	baseConfig := awssdk.Config{
		Region:       "us-east-1",
		BaseEndpoint: awssdk.String("https://aws.mock"),
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
					q, err := url.ParseQuery(string(bd))
					assert.NoError(t, err)

					switch receivedCalls {
					case 1:
						assert.Equal(t, "AssumeRoleWithWebIdentity", q.Get("Action"))
						assert.Equal(t, globalRoleARN, q.Get("RoleArn"))
						assert.Equal(t, tokenFileContent, q.Get("WebIdentityToken"))
						return middleware.FinalizeOutput{
							Result: &sts.AssumeRoleWithWebIdentityOutput{
								Credentials: &types.Credentials{
									AccessKeyId:     awssdk.String("AKIAFAKEEXAMPLE00001"),
									SecretAccessKey: awssdk.String("FAKEwJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY1"),
									SessionToken:    awssdk.String("FwoGZXIvYXdzEFAaDFAKESESSIONTOKENEXAMPLE1"),
									Expiration:      awssdk.Time(time.Now().Add(defaultIntermediateDuration)),
								},
							},
						}, middleware.Metadata{}, nil
					case 2:
						assert.Equal(t, "AssumeRole", q.Get("Action"))
						assert.Equal(t, remoteRoleARN, q.Get("RoleArn"))
						assert.Equal(t, AWSFormatExternalID(resourceID, externalIDPart), q.Get("ExternalId"))
						return middleware.FinalizeOutput{
							Result: &sts.AssumeRoleOutput{
								Credentials: &types.Credentials{
									AccessKeyId:     awssdk.String("AKIAFAKEEXAMPLE00002"),
									SecretAccessKey: awssdk.String("FAKEwJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY2"),
									SessionToken:    awssdk.String("FwoGZXIvYXdzEFAaDFAKESESSIONTOKENEXAMPLE2"),
									Expiration:      awssdk.Time(time.Now().Add(30 * time.Minute)),
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

	chain := []AWSRoleChainingStep{
		&AWSWebIdentityRoleStep{
			RoleARN:              globalRoleARN,
			WebIdentityTokenFile: tokenPath,
			Options: func(o *stscreds.WebIdentityRoleOptions) {
				o.Duration = defaultIntermediateDuration
			},
		},
		&AWSAssumeRoleStep{
			RoleARN: remoteRoleARN,
			Options: func(aro *stscreds.AssumeRoleOptions) {
				aro.ExternalID = awssdk.String(AWSFormatExternalID(resourceID, externalIDPart))
			},
		},
	}

	result := AWSConfigRoleChaining(baseConfig, chain)
	require.NotNil(t, result.Credentials)

	crd, err := result.Credentials.Retrieve(t.Context())
	require.NoError(t, err)
	require.NotNil(t, crd)
	require.Equal(t, 2, receivedCalls)
}

func TestAWSNewIRSAChainRejectsLongDuration(t *testing.T) {
	_, err := AWSNewIRSAChain(t.Context(), AWSIRSAChainConfig{
		GlobalRoleARN:      "arn:aws:iam::999999999999:role/elastic-global-role",
		RemoteRoleARN:      "arn:aws:iam::123456789012:role/customer-role",
		AssumeRoleDuration: 2 * time.Hour,
	})
	require.ErrorContains(t, err, "assume role duration cannot exceed 1h")
}

func TestAWSNewOIDCChainRejectsLongDuration(t *testing.T) {
	_, err := AWSNewOIDCChain(t.Context(), AWSOIDCChainConfig{
		JWTFilePath:        "/path/to/token",
		GlobalRoleARN:      "arn:aws:iam::999999999999:role/elastic-global-role",
		RemoteRoleARN:      "arn:aws:iam::123456789012:role/customer-role",
		AssumeRoleDuration: 2 * time.Hour,
	})
	require.ErrorContains(t, err, "assume role duration cannot exceed 1h")
}
