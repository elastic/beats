// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	azureidentity "github.com/elastic/beats/v7/x-pack/libbeat/common/azure/identity"
	"github.com/elastic/elastic-agent-libs/logp"
)

// addAzureADWebIdentityCredentials configures the AWS config to authenticate
// using an Azure AD token via STS AssumeRoleWithWebIdentity.
func addAzureADWebIdentityCredentials(config ConfigAWS, awsConfig *awssdk.Config, logger *logp.Logger) error {
	logger = logger.Named("addAzureADWebIdentityCredentials")
	logger.Debug("Switching credentials provider to Azure AD web identity")

	tokenProvider, err := azureidentity.NewTokenProvider(config.AzureAD)
	if err != nil {
		return err
	}

	provider := stscreds.NewWebIdentityRoleProvider(
		sts.NewFromConfig(*awsConfig),
		config.RoleArn,
		tokenProvider,
		func(opt *stscreds.WebIdentityRoleOptions) {
			if config.AssumeRoleDuration > 0 {
				opt.Duration = config.AssumeRoleDuration
			}
		},
	)

	awsConfig.Credentials = awssdk.NewCredentialsCache(provider, func(options *awssdk.CredentialsCacheOptions) {
		if config.AssumeRoleExpiryWindow > 0 {
			options.ExpiryWindow = config.AssumeRoleExpiryWindow
		}
	})

	return nil
}
